# goPlexr

PLEX Super Duper Finder — find duplicate media versions in a Plex server and generates JSON/HTML reports.
goPlexr queries a Plex server for duplicate items (movies or optionally shows), optionally performs a deep fetch to inspect media/part details (file paths, sizes) and can verify whether files exist on disk. It summarizes duplicates per library and emits a machine-friendly JSON structure (console, suppressable, or to file) or a standalone HTML report.

## Features
- Discover movie libraries (and optionally show libraries) on a Plex server using the plex server API.
- Detect items with multiple media versions (e.g., 1080p + 4K, different containers/codecs).
- Optionally deep-fetch each item to get Media/Part details (file paths, sizes).
- Optionally verify files on disk (adds Plex's checkFiles=1 to the request).
- Apply a configurable duplicate policy: by default it ignores exact 4K+1080 pairs to avoid flagging intentional duplicates.
- Output JSON to stdout (and/or write to file) and render a self-contained HTML report.

## Quick examples
Run a scan and print JSON to stdout:

```bash
./goplexr -url http://plex-host:32400 -token YOUR_PLEX_TOKEN
```

Write JSON to a file and also create an HTML report:

```bash
./goplexr -url http://plex-host:32400 -token YOUR_PLEX_TOKEN -json-out report.json -html-out report.html
```

Run without verification (faster, but ghost/missing-file counts will be zero):

```bash
./goplexr -url http://plex-host:32400 -token YOUR_PLEX_TOKEN -verify=false
```

Scan only specific library section IDs (skip discovery):

```bash
./goplexr -url http://plex-host:32400 -token YOUR_PLEX_TOKEN -sections 1,2,3
```

## Building

This repository is written in Go. Build a static binary (or for your platform) with the Go toolchain.

From the repository root:

```bash
go build -o goplexr ./cmd/goPlexr
```

Optionally set a version at build time (embedded into the binary):

```bash
go build -ldflags "-X main.Version=v0.3.0" -o goplexr ./cmd/goPlexr
```

Requirements:

- Go 1.20+ (or a reasonably recent Go toolchain).

## Usage and CLI options

Basic usage:

```text
goPlexr -url http://HOST:32400 -token TOKEN [options]
```

Primary flags (defaults shown where applicable):

- -url string
	- Plex base URL (e.g. http://HOST:32400). Can also be set with the environment variable `PLEX_URL`.
- -token string
	- Plex X-Plex-Token. Can also be set with the environment variable `PLEX_TOKEN`.
- -sections string
	- Comma-separated section IDs to scan. When set, auto-discovery is skipped and only these section IDs are processed.
- -include-shows (bool, default: false)
	- Also scan libraries of type `show` in addition to `movie` libraries.
- -deep (bool, default: true)
	- Perform a deep fetch per item to obtain Media/Part details (file path, size). Deep fetch is required to get checkFiles verification.
- -verify (bool, default: true)
	- Verify on-disk files (adds `checkFiles=1` to the deep fetch). Slower but yields accurate 'ghost' counts.
- -pretty (bool, default: true)
	- Pretty-print JSON output.
- -json-out string
	- Write JSON output to this file in addition to stdout (use `-quiet` to disable stdout).
- -html-out string
	- Write a standalone HTML report to this file.
- -quiet / -q (bool)
	- Do not write JSON to stdout; use file outputs instead.
- -insecure (bool)
	- Skip TLS verification (useful for self-signed Plex HTTPS endpoints).
- -timeout duration (default: 20s)
	- HTTP timeout per request.
- -verbose / -V (bool)
	- Verbose logs written to stderr.
- -version / -v (bool)
	- Print version and exit.
- -dup-policy string (default: "ignore-4k-1080")
	- Duplicate policy. Supported values:
		- `ignore-4k-1080` (default): If an item has exactly two versions (one 2160 and one 1080) and no other versions, it will be excluded from duplicate counts and listed under "Ignored" in the report.
		- `plex`: Count any multi-version item as duplicates (Plex-like behavior).

Notes on flags:

- Flags may also be passed with `--long` style (e.g. `--url`) — the CLI normalizes double-dash to single-dash automatically.
- If `-sections` is omitted, the tool will auto-discover libraries on the server and scan all movie libraries (and shows if `-include-shows` is set).

## Output format

goPlexr emits a JSON object (to stdout by default) representing the scan. Top-level fields include:

- server: the Plex base URL used.
- sections: array of `SectionResult` objects; each contains `section_id`, `section_title`, `type`, and `items` (duplicate items only).
- total_duplicate_items, total_versions, total_ghost_parts: summary numbers.
- summary: aggregation with per-library `libraries` summaries and `duplicate_policy` used.
- ignored: optional list of items excluded by the duplicate policy (e.g., exact 4K+1080 pairs).

The HTML report (if written with `-html-out`) is a single self-contained file with an interactive summary and per-item details, and will show badges for verification status when `-verify` is enabled.

## Example workflow

1. Run a full verification and write both JSON and HTML:

```bash
./goplexr -url http://plex:32400 -token TOKEN -html-out report.html -json-out report.json
```

2. Review `report.html` in a browser for an easy human summary.
3. Use `report.json` as input for automation (alerts, dashboards, or retention scripts).

## How duplicate decisions are made

- By default the tool uses the `ignore-4k-1080` policy which ignores items where the only two versions are one 2160 (4K) and one 1080p. This avoids flagging many intentional duplicates where a remux and a 4K are both kept.
- When `-dup-policy=plex` the tool counts any item with multiple versions as duplicates.

Resolution detection is implemented by inspecting the Media.VideoResolution attribute and falling back to dimensions (width/height) when necessary.

## Exit codes

- 0: success
- non-zero: fatal errors (e.g., missing required flags, HTTP errors). Error messages are printed to stderr.

## Development / Contributing

Small, focused PRs are welcome. Areas that may be of interest:

- Add more duplicate policies or heuristics.
- Improve performance when scanning very large libraries.
- Add unit tests around resolution detection and policy behaviour.

Please follow the project's existing code style and run `go vet` / `go test` where applicable before submitting a PR.

## License

MIT — do what you want, just don’t blame me if your cat deletes your library. 😸

## Screens 

HTML Output
<img width="1895" height="2518" alt="html-output-sample" src="https://github.com/user-attachments/assets/f3b4b35a-7290-4191-aa7d-d8f52957e42f" />

JSON Output
<img width="1895" height="2638" alt="json-output-sample" src="https://github.com/user-attachments/assets/144bf00b-17e2-4ee1-b703-36fc48db312b" />







