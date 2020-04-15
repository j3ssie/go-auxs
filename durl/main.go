package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

// diff URLs
// Strip out similar URLs by unique hostname-path-paramName

func main() {
	data := make(map[string]string)
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if sc.Err() != nil && raw == "" {
			continue
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
