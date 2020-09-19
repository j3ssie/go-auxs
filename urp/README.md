## URLs Replace

Parse URLs in fuzz format

## Install

```
go get -u github.com/j3ssie/go-auxs/urp
```

## Usage

```shell
echo 'https://sub.target.com/foo/bar.php?order_id=0018200' | urp

https://sub.target.com/foo/bar.php?order_id=FUZZ
https://sub.target.com/FUZZ?order_id=0018200
https://sub.target.com/foo/FUZZ?order_id=0018200
```