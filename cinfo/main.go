package main

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/genkiroid/cert"
)

// Extract domain from SSL info
// cat /tmp/list_of_IP | cinfo -c 100
var (
	verbose bool
	alexa   bool
)

func main() {
	// cli aguments
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&alexa, "a", false, "Check Alexa Rank of domain")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Strings(args)
		url := args[len(args)-1]

		hostname := getHostName(url, "")
		if !getCerts(hostname) {
			getCerts(getHostName(hostname, "443"))
		}
		os.Exit(0)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				hostname := getHostName(job, "")
				if hostname != "" {
					if !getCerts(hostname) {
						getCerts(getHostName(job, "443"))
					}
				}
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			url := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && url != "" {
				jobs <- url
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func getHostName(raw string, port string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	var hostname string
	if port != "" {
		return u.Hostname() + ":" + port
	}

	if u.Port() == "" {
		hostname = u.Hostname()
	} else {
		hostname = u.Hostname() + ":" + u.Port()
	}
	return hostname
}

func getCerts(raw string) bool {
	var certs cert.Certs
	var err error
	var rank string

	cert.SkipVerify = true

	certs, err = cert.NewCerts([]string{raw})
	if err != nil {
		return false
	}

	for _, certItem := range certs {
		if verbose {
			info, err := GetCertificatesInfo(raw)
			if err == nil {
				fmt.Printf("%s - %s\n",raw, info)
			}
		}

		for _, domain := range certItem.SANs {
			if alexa {
				rank, _ = getAlexaRank(domain)
				fmt.Printf("%v,%v,%s\n", raw, domain, rank)
			} else {
				fmt.Printf("%v,%v\n", raw, domain)
			}
		}
	}
	return true

}

func getAlexaRank(raw string) (string, error) {
	rank := "-1"

	resp, err := http.Get("http://data.alexa.com/data?cli=10&dat=snbamz&url=" + raw)
	if err != nil {
		return rank, err
	}

	defer resp.Body.Close()

	alexaData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rank, err
	}

	decoder := xml.NewDecoder(strings.NewReader(string(alexaData)))
	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}

		switch startElement := token.(type) {
		case xml.StartElement:
			if startElement.Name.Local == "POPULARITY" {
				if len(startElement.Attr) >= 2 {
					rank = startElement.Attr[1].Value
				}
			}
		}
	}
	return rank, nil
}

func GetCertificatesInfo(address string) (string, error) {
	conn, err := tls.Dial("tcp", address, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return fmt.Sprintf("%v", conn.ConnectionState().PeerCertificates[0].Subject), nil
}
