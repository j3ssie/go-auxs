# Old Urls
Fetch known URLs from AlienVault's [Open Threat Exchange](https://otx.alienvault.com), the Wayback Machine, and Common Crawl. Originally built as a microservice.

Copied from my friend [theblackturtle](https://github.com/theblackturtle) repo

### Usage:
```
echo 'example.com' | ourl
```

or

```
ourl example.com
```

```shell
Usage of ourl:
  -a	print domain only
  -f string
    	Wayback Machine filter (filter=statuscode:200&filter=!mimetype:text/html)
  -p	if the data is large, get by pages
  -r	print raw output (JSON format)
  -subs
    	include subdomains of target domain
  -v	enable verbose
```

### install:
```
GO111MODULE=on go get -u github.com/j3ssie/go-auxs/ourl
```

## Credits:
Thanks @tomnomom for [waybackurls](https://github.com/tomnomnom/waybackurls)!
Thanks @lc for gau!
