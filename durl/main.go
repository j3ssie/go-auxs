package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Strip out similar URLs by unique hostname-path-paramName
// cat urls.txt | durl
// only grep url have parameter
// cat urls.txt | durl -p

var (
	blacklist bool
	haveParam bool
	ext       string
)

func main() {
	// cli aguments
	flag.BoolVar(&blacklist, "b", true, "Enable blacklist")
	flag.BoolVar(&haveParam, "p", false, "Enable check if input have parameter")
	flag.StringVar(&ext, "e", "", "Blacklist regex string (default is static extentions)")
	flag.Parse()

	// default blacklist
	if ext == "" {
		ext = `(?i)\.(png|apng|bmp|gif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf|css)(?:\?|#|$)`
	}

	data := make(map[string]string)
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if sc.Err() != nil && raw == "" {
			continue
		}

		if blacklist {
			if IsBlacklisted(raw) {
				continue
			}
		}

		hash := hashUrl(raw)
		if hash == "" {
			continue
		}
		_, exist := data[hash]
		if !exist {
			data[hash] = raw
			fmt.Println(raw)
		}
	}
}

// IsBlacklisted check if url is blacklisted or not
func IsBlacklisted(raw string) bool {
	r, err := regexp.Compile(ext)
	if err != nil {
		return false
	}
	isBlacklisted := r.MatchString(raw)
	if isBlacklisted {
		return true
	}

	// check if have param
	if haveParam {
		p, _ := regexp.Compile(`\?.*\=`)
		return !p.MatchString(raw)
	}

	return false
}

// hashUrl gen unique hash base on url
func hashUrl(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	var queries []string
	for k := range u.Query() {
		queries = append(queries, k)
	}
	sort.Strings(queries)
	query := strings.Join(queries, "-")
	data := fmt.Sprintf("%v-%v-%v", u.Hostname(), u.Path, query)
	return genHash(data)
}

// genHash gen SHA1 hash from string
func genHash(text string) string {
	h := sha1.New()
	h.Write([]byte(text))
	hashed := h.Sum(nil)
	return fmt.Sprintf("%v", hashed)
}
