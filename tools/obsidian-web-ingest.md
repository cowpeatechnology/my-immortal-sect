# URL -> Obsidian Web Ingest

This tool fetches a webpage, extracts the main content, asks the local Codex OAuth-backed model session to organize it, and writes the result into an Obsidian vault.

## Script

- `tools/url_to_obsidian.py`

## What It Does

1. Fetches a URL or `file://` URL
2. Extracts readable content with `lxml`
3. Uses the local Codex OAuth session from `~/.codex/auth.json`
4. Writes a Markdown note into an Obsidian vault
5. Optionally opens the note in Obsidian

## Why It Writes Files Directly

The script writes directly into the vault by default because it works even if the Obsidian CLI is not installed on the current machine.

You can still ask the script to open the note in Obsidian:

- with the Obsidian CLI if it is installed
- otherwise with the official `obsidian://open?...` URI path

## Quick Start

```bash
python3 tools/url_to_obsidian.py \
  "https://example.com/article" \
  --vault "$OBSIDIAN_VAULT_PATH"
```

Put notes into a specific folder:

```bash
python3 tools/url_to_obsidian.py \
  "https://example.com/article" \
  --vault "/absolute/path/to/MyVault" \
  --folder "Inbox/Web Research"
```

Open the resulting note in Obsidian:

```bash
python3 tools/url_to_obsidian.py \
  "https://example.com/article" \
  --vault "/absolute/path/to/MyVault" \
  --open-note
```

Prefer the Obsidian CLI when it is installed:

```bash
python3 tools/url_to_obsidian.py \
  "https://example.com/article" \
  --vault "/absolute/path/to/MyVault" \
  --open-note \
  --prefer-cli
```

Dry-run the extraction without calling the model:

```bash
python3 tools/url_to_obsidian.py \
  "file:///tmp/sample.html" \
  --vault "/absolute/path/to/MyVault" \
  --skip-llm
```

## Environment

- Optional: `OBSIDIAN_VAULT_PATH=/absolute/path/to/vault`

## Related References

- Obsidian CLI: `https://obsidian.md/help/cli`
- Obsidian URI: `https://help.obsidian.md/uri`
- OAuth helper reused conceptually from:
  - `tools/gpt_image_1_5_codex_oauth.py`
