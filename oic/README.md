## Open in Chrome
Open URL with your real browser


## Install

```bash
go get -u github.com/j3ssie/go-auxs/oic
```

## Usage

```bash
cat urls.txt | oic
cat urls.txt | oic -c 5 -proxy http://127.0.0.1:8080
cat urls.txt | oic -c 5 -proxy http://127.0.0.1:8080 -q
```