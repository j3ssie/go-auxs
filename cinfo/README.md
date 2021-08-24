## Cinfo

Extract domain from SSL info

## Install

```shell
GO111MODULE=on go get -u github.com/j3ssie/go-auxs/cinfo
```

## Usage

```shell
# Basic Usage
echo '1.2.3.4:443' | cinfo
echo '1.2.3.4:443' | cinfo -json


# probe for common SSL ports like 443,8443
echo '1.2.3.4' | cinfo -e

# get alexa rank of domains
echo '1.2.3.4' | cinfo -e -a
```