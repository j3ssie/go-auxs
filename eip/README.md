## Extend IP

Extend the IP range by CIDR

## Install

```
go get -u github.com/j3ssie/go-auxs/eip
```

## Usage
```
# Basic usage
echo '1.2.3.4/24' | eip

# Append common port to ip

echo '1.2.3.4/24' | eip -p s
```