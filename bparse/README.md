# bparse

Parse Burp Suite XML exports (Proxy history → Save items). Streams the file one item at a time, so large exports are fine. No dependencies.

## Build

```
go build -o bparse .
```

## Usage

```
bparse [flags] burp.xml        # export CSV
bparse stat [flags] burp.xml   # statistics
```

Flags can go before or after the file name.

| Flag | Meaning |
|------|---------|
| `-i file` | Burp file (default: first positional argument) |
| `-o file` | Output file (default `out.csv`, or `out.txt` with `-f`) |
| `-d` | CSV: write bodies as decoded text instead of base64 |
| `-n` | CSV: leave out request/response bodies |
| `-r` | Print every request/response fully decoded to stdout (for grep) |
| `-f` | Write raw requests, base64, one per line |
| `-top N` | `stat`: number of top hosts/URLs (default 10) |

## Modes

**CSV (default)**: columns `URL, Method, Host, Path, Protocol, IP:Port, Status, Responselength, Mimetype, RequestBody, ResponseBody`. Bodies are base64 unless `-d`, which gives readable text (chunked and gzip undone) but may produce multi-line cells, and garbage for binary bodies.

**`-r`**: plain-text dump, nothing is written to disk:

```
bparse burp.xml -r | grep -i authorization
bparse burp.xml -r > all.txt
```

Each item starts with `=== #N METHOD URL [status] ===`, followed by `--- request ---` and `--- response ---`. Headers are unchanged (a gzipped response still shows `Content-Encoding: gzip`); bodies are decoded.

**`-f`**: one base64-encoded raw request per line, for tools that want one request per line. Decode a line with `base64 -d`.

**`stat`**: totals (requests, responses, unique hosts/URLs, response bytes) and counts by method, status, protocol, MIME type, plus top hosts and URLs. Memory grows with the number of unique URLs.

## Notes

- Malformed HTTP messages fall back to splitting headers and body at the first blank line.
- `-d`, `-n` and `-top` do nothing in modes where they don't apply.
