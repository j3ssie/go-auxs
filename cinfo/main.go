package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/genkiroid/cert"
)

// Extract domain from SSL info
// cat /tmp/list_of_IP | cinfo -c 100
var verbose bool

func main() {
	// cli aguments
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Sort(sort.StringSlice(args))
		url := args[len(args)-1]

		hostname := getHostName(url)
		getCerts(hostname)
		os.Exit(0)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				hostname := getHostName(job)
				if hostname != "" {
					getCerts(hostname)
				}
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			url := strings.TrimSpace(sc.Text())
			jobs <- url
		}
		close(jobs)
	}()
	wg.Wait()

}

func getHostName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		raw = "https://" + raw
		u, err = url.Parse(raw)
		if err != nil {
			return ""
		}
	}
	var hostname string
	if u.Port() == "" {
		hostname = u.Hostname()
	} else {
		hostname = u.Hostname() + ":" + u.Port()
	}
	return hostname
}

func getCerts(url string) bool {
	var certs cert.Certs
	var err error
	cert.SkipVerify = true

	certs, err = cert.NewCerts([]string{url})
	if err != nil {
		return false
	}

	for _, certItem := range certs {
		if verbose == true {
			fmt.Printf("%s", certs)
		} else {
			for _, domain := range certItem.SANs {
				fmt.Printf("%v,%v\n", url, domain)
			}
		}
	}
	return true

}
