package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/markbates/pkger"
	"github.com/panjf2000/ants"
	"github.com/thoas/go-funk"
	"golang.org/x/net/publicsuffix"
	"log"
	"net"

	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// public suffix finder
// cat /tmp/list_of_IP | psuff -c 100

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "any string"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	verbose              bool
	extra                bool
	org                  string
	concurrency          int
	publicSuffixContents []string
	prefixes             arrayFlags
	domains              arrayFlags
)

var logger *log.Logger

func main() {
	// cli aguments
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.StringVar(&org, "org", "", "Specific Org")
	flag.Var(&prefixes, "p", "prefix domains")
	flag.Var(&domains, "d", "input domains")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	// prepare suffix data
	// update: wget https://publicsuffix.org/list/public_suffix_list.dat
	// pkger will discover that we need example.txt and embed it
	f, err := pkger.Open("/public_suffix_list.dat")
	if err != nil {
		panic(err)
	}
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	ParseSuffix(string(contents))

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Strings(args)
		raw := args[len(args)-1]
		domains = append(domains, raw)
	}

	// really start to do something
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(concurrency, func(i interface{}) {
		defer wg.Done()
		job := i.(string)

		line := getOrg(job)
		log.Printf("Parsed org: %s\n", line)
		doResolving(line)
	}, ants.WithPreAlloc(true))
	defer p.Release()

	// reading input

	// detect if anything came from std
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			raw := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && raw != "" {
				domains = append(domains, raw)
			}
		}
	}
	domains = funk.UniqString(domains)
	if len(domains) > 0 {
		for _, domain := range domains {
			wg.Add(1)
			raw := strings.TrimSpace(domain)
			log.Printf("Parsed processing: %s\n", raw)
			p.Invoke(raw)
		}
	}
	wg.Wait()
}

func doResolving(org string) {
	var dg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(concurrency, func(i interface{}) {
		defer dg.Done()
		domain := i.(string)
		lookup(domain)

	}, ants.WithPreAlloc(true))
	defer p.Release()

	for _, suffix := range publicSuffixContents {
		dg.Add(1)
		job := fmt.Sprintf("%s.%s", org, suffix)
		p.Invoke(job)

		if len(prefixes) > 0 {
			for _, prefix := range prefixes {
				dg.Add(1)
				job := fmt.Sprintf("%s.%s.%s", prefix, org, suffix)
				p.Invoke(job)
			}
		}
	}
	dg.Wait()
}

func lookup(domain string) {
	if resolved, err := net.LookupHost(domain); err == nil {
		if verbose {
			//resolved = funk.UniqString(resolved)
			for _, ip := range resolved {
				fmt.Printf("%s,%s\n", domain, ip)
			}
			return
		}
		fmt.Println(domain)
	}
}

func ParseSuffix(content string) {
	data := strings.Split(content, "\n")
	for _, line := range data {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "//") {
			continue
		}

		if isASCII(line) {
			publicSuffixContents = append(publicSuffixContents, line)
		}
	}
	log.Printf("Parsed %v suffixes\n", len(publicSuffixContents))
	publicSuffixContents = funk.UniqString(publicSuffixContents)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func getOrg(raw string) string {
	data := ParseTarget(raw)
	if len(data) <= 1 {
		return raw
	}

	return data["Org"]
}

// ParseTarget parsing target and some variable for template
func ParseTarget(raw string) map[string]string {
	target := make(map[string]string)
	if raw == "" {
		return target
	}
	target["Target"] = raw
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
