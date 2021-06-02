## Clean Subdomains

Clean your subdomain list

## Install

```
go get -u github.com/j3ssie/go-auxs/cleansub
```

## Usage
```shell
cat subdomains.txt
target.com
sub1.target.com
aatarget.com
sub2.target.com

cat subdomains.txt | cleansub
target.com
sub1.target.com
sub2.target.com
```