# CDN Filter

Cleaning CDN IP Address and Private IPs from list of inputs

## Install

```shell
go get -u github.com/j3ssie/go-auxs/cdnfilter
```

## Usage

```shell
cat list_of_ips.txt | cdnfilter -c cdn_out.txt -n not_cdn_out.txt
```

## Credit

Created by my friend @thebl4ckturtle
