## URLs Replace

Parse URLs in fuzz format

## Install

```
go get -u github.com/j3ssie/go-auxs/urp
```

## Usage

> **NOTE:** always pass output to unique tools like `sort -u`

### For run dirbscan

```shell
echo 'https://sub.target.com/foo/bar.php?order_id=0018200' | urp

https://sub.target.com/foo/bar.php?order_id=FUZZ
https://sub.target.com/FUZZ?order_id=0018200
https://sub.target.com/foo/FUZZ?order_id=0018200

echo 'https://sub.target.com/foo/bar.php?order_id=0018200' | urp -qq
https://sub.target.com/FUZZ
https://sub.target.com/foo/FUZZ

```

### Get base Paths only (useful when using `jaeles --ba`)

```shell
echo 'http://sub.target.com:80/34032/cegetel/fr-fr/index.php?q=123' | urp -I '' -qq

http://sub.target.com
http://sub.target.com/34032/
http://sub.target.com/34032/cegetel/
http://sub.target.com/34032/cegetel/fr-fr/


cat urls.txt | urp -I '' -qq | sort -u

```
