# agentize

A small CLI that fetches a master `AGENTS.md` from a GitHub Gist and places it in the current directory.

Edits to the gist take effect on the next run, without rebuilding the binary.

## Features

- Fetches `AGENTS.md` from a GitHub Gist at runtime
- Registers the gist once with `agentize init`
- Opens the registered gist in your browser with `agentize open`
- Adds `AGENTS.md` to `.gitignore` automatically (skippable)
- Refuses to overwrite an existing `AGENTS.md` unless `-force` is given
- Uses only the Go standard library

## Install

```cmd
go install github.com/AWtnb/agentize@latest
```

Or build from source:

```cmd
git clone https://github.com/AWtnb/agentize.git
cd agentize
go install
```

## Setup

Register your gist once:

```cmd
agentize init https://gist.github.com/<user>/<gist-id>
```

The URL is validated by fetching it, then saved to `~/.agentize` (`%USERPROFILE%\.agentize` on Windows).

If your gist has multiple files, use the raw URL of the file to pin it:

```cmd
agentize init https://gist.githubusercontent.com/<user>/<gist-id>/raw/AGENTS.md
```

Only `gist.github.com` and `gist.githubusercontent.com` are accepted.

## Usage

Write `AGENTS.md` to the current directory:

```cmd
agentize
```

Overwrite an existing `AGENTS.md`:

```cmd
agentize -force
```

Do not touch `.gitignore`:

```cmd
agentize -no-ignore
```

Open the registered gist in your browser (for editing):

```cmd
agentize open
```

## Commands

| Command | Description |
|---|---|
| `agentize init <gist-url>` | Validate the gist URL and save it to `~/.agentize` |
| `agentize open` | Open the registered gist page in the default browser |
| `agentize` | Fetch the gist and write `AGENTS.md` to the current directory |

## Options

These options apply to `agentize` (without a subcommand).

| Option | Description |
|---|---|
| `-force` | Overwrite an existing `AGENTS.md` |
| `-no-ignore` | Do not update `.gitignore` |

## Workflow

```plaintext
agentize open   ->  edit AGENTS.md on the gist
agentize        ->  apply the latest content to the current project
```

## Notes

- **Multi-file gists**: the gist page URL resolves to the first file. Pin a file with its raw URL (see Setup).
- **Secret gists**: anyone with the URL can read the content. Do not put sensitive information in it.
- **Caching**: raw URLs may be cached for a few minutes, so a fresh edit can take a moment to appear.
- **Offline**: fetching fails without a network connection.
- **Network timeout**: requests time out after 15 seconds.

## Files

| Path | Purpose |
|---|---|
| `~/.agentize` | Registered gist URL (one line) |
| `./AGENTS.md` | Generated file in the current directory |
| `./.gitignore` | Updated to ignore `AGENTS.md` |

## License

MIT