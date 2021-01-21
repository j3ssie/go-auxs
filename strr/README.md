## strr

String replace

## Install

```
go get -u github.com/j3ssie/go-auxs/strr
```

## Usage

```shell
echo 'domain.com' | strr -t '{}.{{.Raw}}' -I wordlists.txt

echo 'www-{}.domain.com' | strr -I wordlists.txt

cat domains.txt | strr -t '{}.{{.Raw}}' -i 'dev'
```