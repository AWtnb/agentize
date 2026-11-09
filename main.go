package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	target     = "AGENTS.md"
	configName = ".agentize"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if 0 < len(args) && args[0] == "init" {
		return runInit(args[1:])
	}
	if 0 < len(args) && args[0] == "open" {
		return runOpen()
	}
	return runApply(args)
}

// runOpen opens the registered gist page in the default browser.
func runOpen() error {
	gistURL, err := loadGistURL()
	if err != nil {
		return err
	}
	pageURL, err := toGistPageURL(gistURL)
	if err != nil {
		return err
	}
	if err := openBrowser(pageURL); err != nil {
		return err
	}
	fmt.Printf("Opened: %s\n", pageURL)
	return nil
}

// toGistPageURL converts a raw content URL into the gist page URL.
// Page URLs are returned with the /raw suffix removed.
func toGistPageURL(input string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}

	switch u.Host {
	case "gist.github.com":
		path := strings.TrimSuffix(u.Path, "/")
		path = strings.TrimSuffix(path, "/raw")
		u.Path = path
		u.RawQuery = ""
		u.Fragment = ""
		return u.String(), nil
	case "gist.githubusercontent.com":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", errors.New("invalid raw gist URL")
		}
		return "https://gist.github.com/" + parts[0] + "/" + parts[1], nil
	}
	return "", errors.New("URL host must be gist.github.com or gist.githubusercontent.com")
}

// openBrowser opens the URL in the default browser of the current OS.
func openBrowser(target string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	case "darwin":
		return exec.Command("open", target).Start()
	}
	return exec.Command("xdg-open", target).Start()
}

// runInit validates the gist URL and saves it to the config file.
func runInit(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: agentize init <gist-url>")
	}

	rawURL, err := normalizeGistURL(args[0])
	if err != nil {
		return err
	}
	if _, err := fetch(rawURL); err != nil {
		return fmt.Errorf("failed to fetch gist: %w", err)
	}

	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(args[0]+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Saved: %s\n", path)
	return nil
}

// runApply fetches the gist content and writes AGENTS.md to the current directory.
func runApply(args []string) error {
	fs := flag.NewFlagSet("agentize", flag.ContinueOnError)
	force := fs.Bool("force", false, "Overwrite existing files")
	noIgnore := fs.Bool("no-ignore", false, "Do not update .gitignore")
	if err := fs.Parse(args); err != nil {
		return err
	}

	gistURL, err := loadGistURL()
	if err != nil {
		return err
	}
	rawURL, err := normalizeGistURL(gistURL)
	if err != nil {
		return err
	}
	data, err := fetch(rawURL)
	if err != nil {
		return fmt.Errorf("failed to fetch gist: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := writeAgents(filepath.Join(cwd, target), data, *force); err != nil {
		return err
	}
	if *noIgnore {
		return nil
	}
	return ensureIgnored(filepath.Join(cwd, ".gitignore"))
}

// configPath returns the path of the config file in the home directory.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configName), nil
}

// loadGistURL reads the saved gist URL from the config file.
func loadGistURL() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("not initialized (run: agentize init <gist-url>)")
	}
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("%s is empty (run: agentize init <gist-url>)", path)
	}
	return value, nil
}

// normalizeGistURL converts a gist page URL into a raw content URL.
// Raw URLs are returned as is.
func normalizeGistURL(input string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" {
		return "", errors.New("gist URL must start with https://")
	}

	switch u.Host {
	case "gist.githubusercontent.com":
		return u.String(), nil
	case "gist.github.com":
		path := strings.TrimSuffix(u.Path, "/")
		if strings.Count(path, "/") < 2 {
			return "", errors.New("invalid gist URL (expected https://gist.github.com/<user>/<id>)")
		}
		if strings.HasSuffix(path, "/raw") {
			return u.String(), nil
		}
		u.Path = path + "/raw"
		u.RawQuery = ""
		u.Fragment = ""
		return u.String(), nil
	}
	return "", errors.New("URL host must be gist.github.com or gist.githubusercontent.com")
}

// fetch downloads the content of the given URL.
func fetch(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func writeAgents(dest string, data []byte, force bool) error {
	if _, err := os.Stat(dest); err == nil && !force {
		return fmt.Errorf("%s already exists (use -force to overwrite)", target)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote: %s\n", dest)
	return nil
}

func ensureIgnored(path string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if isIgnored(data) {
		fmt.Println(".gitignore: already registered; no changes made")
		return nil
	}

	var buf bytes.Buffer
	buf.Write(data)
	if 0 < len(data) && !bytes.HasSuffix(data, []byte("\n")) {
		buf.WriteString("\n")
	}
	buf.WriteString(target + "\n")

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("Added %s to .gitignore\n", target)
	return nil
}

func isIgnored(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == target || line == "/"+target {
			return true
		}
	}
	return false
}
