## Extend IP

Extend the IP range by CIDR

## Install

```
go get -u github.com/j3ssie/go-auxs/eip
```

## Usage
```shell
# Basic usage
echo '1.2.3.4/20' | eip -s 24

# Append common port to ip

echo '1.2.3.4/24' | eip -p s
```