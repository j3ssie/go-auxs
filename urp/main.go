package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	path2 "path"
	"sort"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
)

const (
	ReplaceAll      = "all"
	ReplaceOneByOne = "one-by-one"
)

var (
	IgnoreExtensions = []string{"png", "apng", "bmp", "gif", "ico", "cur", "jpg", "jpeg", "jfif", "pjp", "pjpeg", "svg", "tif", "tiff", "webp", "xbm", "3gp", "aac", "flac", "mpg", "mpeg", "mp3", "mp4", "m4a", "m4v", "m4p", "oga", "ogg", "ogv", "mov", "wav", "webm", "eot", "woff", "woff2", "ttf", "otf", "css"}
)

// Literally copied from: https://github.com/theblackturtle/ureplace
// with some improvements

var (
	appendMode      bool
	query           bool
	path            bool
	removeMediaExt  bool
	removeLastPath  bool
	last            bool
	paramName       bool
	place           string
	blacklistExt    string
	toInjectList    string
	injectWords     string
	InjectAll       bool
	RemoveQuery     bool
	TrimLastSlash   bool
	RemoveDummyPort bool
	payloadList     []string
)

func main() {
	flag.BoolVar(&appendMode, "a", false, "Append the value")
	flag.BoolVar(&removeMediaExt, "m", false, "Ignore media extensions")
	flag.BoolVar(&query, "n", false, "Inject payload to param name too")
	flag.BoolVar(&paramName, "l", false, "Append payload after the extension")
	flag.BoolVar(&path, "p", false, "Path only (default will replace both path and query)")
	flag.BoolVar(&last, "L", false, "Append payload after the extension")
	// remove some path
	flag.BoolVar(&removeLastPath, "pp", true, "Remove last path")
	flag.BoolVar(&RemoveDummyPort, "ppp", true, "Remove dummy port like :80")
	flag.BoolVar(&TrimLastSlash, "ss", false, "Trim Last Slash")
	flag.BoolVar(&RemoveQuery, "qq", false, "Remove Query String (useful when do dirbscan)")

	flag.StringVar(&blacklistExt, "b", "", "Additional blacklist extensions (Ex: js,html)")
	// new one
	flag.StringVar(&place, "i", "one-by-one", "Where to inject (when using with path or value)\n  all: replace all\n  one: replace one by one\n  2: replace the second path/param\n  -2: replace the second path/param from the end")
	flag.BoolVar(&InjectAll, "A", true, "Inject All")
	flag.StringVar(&injectWords, "I", "FUZZ", "Inject Words to replace")
	flag.StringVar(&toInjectList, "iL", "", "Payload list")
	flag.Parse()

	// prepare words
	payloadList = append(payloadList, injectWords)
	if toInjectList != "" {
		pf, err := os.Open(toInjectList)
		if err != nil {
			panic(err)
		}
		defer pf.Close()
		payloadList = []string{}
		scPayload := bufio.NewScanner(pf)
		for scPayload.Scan() {
			line := strings.TrimSpace(scPayload.Text())
			if line != "" {
				payloadList = append(payloadList, line)
			}
		}
	}

	if blacklistExt != "" {
		bl := strings.Split(blacklistExt, ",")
		for _, e := range bl {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			IgnoreExtensions = append(IgnoreExtensions, e)
		}
	}
	sort.Strings(IgnoreExtensions)

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())


		if RemoveDummyPort {
			raw = strings.Replace(raw, ":80/", "/", -1)
		}

		u, err := url.Parse(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse url %s [%s]\n", sc.Text(), err)
			continue
		}

		if removeMediaExt {
			if BlacklistExt(u) {
				continue
			}
		}

		// really start to do something
		for _, payload := range payloadList {
			var finalUrls []string

			switch {
			case query:
				urls, err := QueryBuilder(u.String(), payload)
				finalUrls = append(finalUrls, urls...)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[QUERY] Failed to generate %s with the payload %s\n", u.String(), payload)
					continue
				}
			case path:
				urls, err := PathBuilder(u.String(), payload)
				finalUrls = append(finalUrls, urls...)

				if err != nil {
					fmt.Fprintf(os.Stderr, "[PATH] Failed to generate %s with the payload %s\n", u.String(), payload)
					continue
				}
			default:
				// query
				if !RemoveQuery {
					urls, err := QueryBuilder(u.String(), payload)
					finalUrls = append(finalUrls, urls...)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[QUERY] Failed to generate %s with the payload %s\n", u.String(), payload)
						continue
					}
				}

				// path
				urls, err := PathBuilder(u.String(), payload)
				finalUrls = append(finalUrls, urls...)

				if err != nil {
					fmt.Fprintf(os.Stderr, "[PATH] Failed to generate %s with the payload %s\n", u.String(), payload)
					continue
				}
			}

			for _, gU := range finalUrls {
				if TrimLastSlash {
					gU = strings.TrimRight(gU, "/")
				}
				fmt.Println(gU)
			}
		}
	}

}

func QueryBuilder(urlString string, payload string) ([]string, error) {
	pp := make([]string, 0)
	urlList := make([]string, 0)

	u, err := url.Parse(urlString)
	if err != nil {
		return urlList, err
	}

	if len(u.Query()) == 0 {
		return urlList, fmt.Errorf("no query")
	}

	for p := range u.Query() {
		pp = append(pp, p)
	}
	sort.Strings(pp)

	switch place {
	case ReplaceAll:
		qs := url.Values{}
		for param, vv := range u.Query() {
			if appendMode {
				qs.Set(param, vv[0]+payload)
			} else {
				qs.Set(param, payload)
			}
		}
		u.RawQuery = qs.Encode()
		uRawQuery, _ := url.QueryUnescape(u.String())
		urlList = append(urlList, uRawQuery)
	case ReplaceOneByOne:
		for i := 0; i < len(pp); i++ {
			cloneURL := &url.URL{}
			err := copier.Copy(cloneURL, u)
			if err != nil {
				return []string{}, fmt.Errorf("Failed to clone url")
			}
			qs := cloneURL.Query()
			if appendMode {
				qs.Set(pp[i], qs.Get(pp[i])+payload)
			} else {
				qs.Set(pp[i], payload)
			}
			cloneURL.RawQuery = qs.Encode()
			cloneURLRawQuery, _ := url.QueryUnescape(cloneURL.String())
			urlList = append(urlList, cloneURLRawQuery)
		}
	default:
		var toReplacePlace int

		if strings.HasPrefix(place, "-") {
			p, err := strconv.Atoi(place[1:])
			if err != nil {
				p = 0
			}
			toReplacePlace = len(pp[:len(pp)-p])
		} else {
			p, err := strconv.Atoi(place)
			if err != nil {
				// fmt.Fprintf(os.Stderr, "failed to convert \"place\" string to int\n")
				p = 0
			}
			toReplacePlace = p
		}

		if toReplacePlace >= len(pp) {
			toReplacePlace = len(pp) - 1
		}

		qs := u.Query()
		if appendMode {
			qs.Set(pp[toReplacePlace], qs.Get(pp[toReplacePlace])+payload)
		} else {
			qs.Set(pp[toReplacePlace], payload)
		}
		u.RawQuery = qs.Encode()
		uRawQuery, _ := url.QueryUnescape(u.String())
		urlList = append(urlList, uRawQuery)
	}
	return urlList, nil
}

func PathBuilder(urlString string, payload string) ([]string, error) {
	urlList := make([]string, 0)

	u, err := url.Parse(urlString)
	if err != nil {
		return urlList, err
	}

	if RemoveQuery {
		q := u.Query()
		for k, _ := range u.Query() {
			q.Del(k)
		}
		u.RawQuery = q.Encode()
	}

	path := strings.TrimPrefix(u.EscapedPath(), "/")
	paths := strings.Split(path, "/")

	if len(paths) == 0 {
		return urlList, fmt.Errorf("no paths")
	}

	switch place {
	case ReplaceAll:
		for i := range paths {
			if appendMode {
				paths[i] = paths[i] + payload
			} else {
				paths[i] = payload
			}
		}
		u.Path = strings.Join(paths, "/")
		uRawPath, _ := url.PathUnescape(u.String())
		urlList = append(urlList, uRawPath)
	case ReplaceOneByOne:
		for i := 0; i < len(paths); i++ {

			cloneURL := &url.URL{}
			err := copier.Copy(cloneURL, u)
			if err != nil {
				return []string{}, fmt.Errorf("Failed to clone url")
			}
			pathClone := append(paths[:0:0], paths...)
			if appendMode {
				pathClone[i] = pathClone[i] + payload
			} else {
				pathClone[i] = payload
				// remove last paths after the payload
				if removeLastPath {
					pathClone = pathClone[:i+1]
				}
			}

			cloneURL.Path = strings.Join(pathClone, "/")
			cloneURLRawPath, _ := url.PathUnescape(cloneURL.String())
			urlList = append(urlList, cloneURLRawPath)
		}
	default:
		var toReplacePlace int
		if strings.HasPrefix(place, "-") {
			p, err := strconv.Atoi(place[1:])
			if err != nil {
				// fmt.Fprintf(os.Stderr, "Failed to convert \"place\" string to int\n")
				p = 0
			}
			toReplacePlace = len(paths[:len(paths)-p])
		} else {
			p, err := strconv.Atoi(place)
			if err != nil {
				// fmt.Fprintf(os.Stderr, "Failed to convert \"place\" string to int\n")
				p = 0
			}
			toReplacePlace = p
		}
		if toReplacePlace >= len(paths) {
			toReplacePlace = len(paths) - 1
		}

		if appendMode {
			paths[toReplacePlace] = paths[toReplacePlace] + payload
		} else {
			paths[toReplacePlace] = payload
		}
		u.Path = strings.Join(paths, "/")
		uRawPath, _ := url.PathUnescape(u.String())
		urlList = append(urlList, uRawPath)
	}

	if last {
		cloneURL := &url.URL{}
		err := copier.Copy(cloneURL, u)
		if err != nil {
			return []string{}, fmt.Errorf("Failed to clone url")
		}
		pathClone := append(paths[:0:0], paths...)
		cloneURL.Path = strings.Join(pathClone, "/") + payload
		cloneURLRawPath, _ := url.PathUnescape(cloneURL.String())
		urlList = append(urlList, cloneURLRawPath)

		cloneURL.Path = strings.Join(pathClone, "/") + "?" + payload
		cloneURLRawPath, _ = url.PathUnescape(cloneURL.String())
		urlList = append(urlList, cloneURLRawPath)
	}

	return urlList, nil
}

// Return true if in blacklist
func BlacklistExt(u *url.URL) bool {
	e := strings.TrimPrefix(path2.Ext(u.Path), ".")

	i := sort.Search(len(IgnoreExtensions), func(i int) bool { return e <= IgnoreExtensions[i] })
	if i < len(IgnoreExtensions) && IgnoreExtensions[i] == e {
		return true
	} else {
		return false
	}
}
