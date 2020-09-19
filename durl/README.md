## Diff URLs

Strip out similar URLs by unique hostname-path-paramName

## Install
```
go get -u github.com/j3ssie/go-auxs/durl
```

## Usage
```
cat wayback_urls.txt | durl | tee differ_urls.txt
```