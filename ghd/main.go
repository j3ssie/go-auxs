package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	verbose     bool
	concurrency int

	dorkFile string
	data     string
	dataFile string
)

func main() {
	// cli args
	flag.StringVar(&data, "u", "", "URL to open")
	flag.StringVar(&dataFile, "U", "", "URL file to open")
	flag.StringVar(&dorkFile, "d", "", "Dorks file")
	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.IntVar(&concurrency, "c", 5, "number of tab at a time")
	flag.Parse()

	// detect if anything came from std
	var templateData, urls []string

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			target := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && target != "" {
				urls = append(urls, target)
			}
		}
	}

	// get url input
	if data != "" {
		urls = append(urls, data)
	}
	if dataFile != "" {
		urls = append(urls, ReadingLines(dataFile)...)
	}

	// get dork data
	if dorkFile == "" {
		goPath, ok := os.LookupEnv("GOPATH")
		if ok {
			dorkFile = path.Join(goPath, "src/github.com/j3ssie/go-auxs/ghd/dorks.txt")
		}

		ghDork, ok := os.LookupEnv("GH_DORKS")
		if ok {
			dorkFile = ghDork
		}
	}

	if dorkFile == "" {
		fmt.Fprintf(os.Stderr, "Need to provide dork file via -d dorks.txt \n")
		os.Exit(-1)
	}
	templateData = ReadingLines(dorkFile)

	for _, u := range urls {
		data := ParseURL(u)
		for _, raw := range templateData {
			out := RenderTemplate(raw, data)
			fmt.Println(out)
		}
	}
}

func RenderTemplate(format string, data map[string]string) string {
	// ResolveData resolve template from signature file
	t := template.Must(template.New("").Parse(format))
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		return format
	}
	return buf.String()
}

func ParseURL(raw string) map[string]string {
	target := make(map[string]string)
	if raw == "" {
		return target
	}
	target["Raw"] = raw
	u, err := url.Parse(raw)

	// something wrong so parsing it again
	if err != nil || u.Scheme == "" || strings.Contains(u.Scheme, ".") {
		raw = fmt.Sprintf("https://%v", raw)
		u, err = url.Parse(raw)
		if err != nil {
			return target
		}
		// fmt.Println("parse again")
	}
	var hostname string
	var query string
	port := u.Port()
	// var domain string
	domain := u.Hostname()

	query = u.RawQuery
	if u.Port() == "" {
		if strings.Contains(u.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}

		hostname = u.Hostname()
	} else {
		// ignore common port in Host
		if u.Port() == "443" || u.Port() == "80" {
			hostname = u.Hostname()
		} else {
			hostname = u.Hostname() + ":" + u.Port()
		}
	}

	target["Scheme"] = u.Scheme
	target["Path"] = u.Path
	target["Domain"] = domain

	target["Org"] = domain
	suffix, ok := publicsuffix.PublicSuffix(domain)
	if ok {
		target["Org"] = strings.Replace(domain, fmt.Sprintf(".%s", suffix), "", -1)
	} else {
		if strings.Contains(domain, ".") {
			parts := strings.Split(domain, ".")
			if len(parts) == 2 {
				target["Org"] = parts[0]
			} else {
				target["Org"] = parts[len(parts)-2]
			}
		}
	}

	target["Host"] = hostname
	target["Port"] = port
	target["RawQuery"] = query

	if (target["RawQuery"] != "") && (port == "80" || port == "443") {
		target["URL"] = fmt.Sprintf("%v://%v%v?%v", target["Scheme"], target["Host"], target["Path"], target["RawQuery"])
	} else if port != "80" && port != "443" {
		target["URL"] = fmt.Sprintf("%v://%v:%v%v?%v", target["Scheme"], target["Domain"], target["Port"], target["Path"], target["RawQuery"])
	} else {
		target["URL"] = fmt.Sprintf("%v://%v%v", target["Scheme"], target["Host"], target["Path"])
	}

	uu, _ := url.Parse(raw)
	target["BaseURL"] = fmt.Sprintf("%v://%v", uu.Scheme, uu.Host)
	target["Extension"] = filepath.Ext(target["BaseURL"])

	return target
}

// ReadingLines Reading file and return content as []string
func ReadingLines(filename string) []string {
	var result []string
	file, err := os.Open(filename)
	if err != nil {
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := strings.TrimSpace(scanner.Text())
		if val == "" {
			continue
		}
		result = append(result, val)
	}

	if err := scanner.Err(); err != nil {
		return result
	}
	return result
}
