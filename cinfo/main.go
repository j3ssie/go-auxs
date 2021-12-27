package main

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/genkiroid/cert"
	jsoniter "github.com/json-iterator/go"
)

// Extract domain from SSL info
// cat /tmp/list_of_IP | cinfo -c 100
var (
	verbose     bool
	alexa       bool
	extra       bool
	jsonOutput  bool
	ports       string
	concurrency int
)

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&jsonOutput, "json", false, "Show Output as Json format")
	flag.BoolVar(&alexa, "a", false, "Check Alexa Rank of domain")
	flag.BoolVar(&extra, "e", false, "Append common extra HTTPS port too")
	flag.StringVar(&ports, "p", "443,8443,9443", "Common extra HTTPS port too (default: 443,8443,9443)")
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
					if extra {
						if strings.Contains(hostname, ":") {
							hostname = strings.Split(hostname, ":")[0]
						}
						hostnames := moreHosts(hostname)
						for _, host := range hostnames {
							getCerts(host)
						}
					}
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

func moreHosts(raw string) []string {
	var result []string
	mports := strings.Split(raw, ",")
	for _, mport := range mports {
		result = append(result, fmt.Sprintf("%s:%s", raw, mport))
	}
	return result
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

type CertInfo struct {
	Input   string   `json:"input"`
	Domains []string `json:"domains"`
	Info    string   `json:"info"`
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

	certInfo := CertInfo{
		Input: raw,
	}

	for _, certItem := range certs {
		if verbose {
			info, err := GetCertificatesInfo(raw)
			certInfo.Info = info
			if err == nil {
				if !jsonOutput {
					fmt.Printf("%s - %s\n", raw, info)
				}
			}
		}

		for _, domain := range certItem.SANs {
			data := domain
			if alexa {
				rank, _ = getAlexaRank(domain)
				data = fmt.Sprintf("%v,%v,%s", raw, domain, rank)
			} else if !jsonOutput {
				data = fmt.Sprintf("%v,%v", raw, domain)
			}

			if jsonOutput {
				certInfo.Domains = append(certInfo.Domains, data)
			} else {
				fmt.Println(data)
			}
		}
	}

	if jsonOutput {
		if data, err := jsoniter.MarshalToString(certInfo); err == nil {
			fmt.Println(data)
		}
	}

	return true

}

func getAlexaRank(raw string) (string, error) {
	rank := "-1"

	if strings.Contains(raw, "*.") {
		raw = strings.ReplaceAll(raw, "*.", "")
	}

	// sub.example.com --> example.com
	suffix, ok := publicsuffix.PublicSuffix(raw)
	if ok {
		root := strings.ReplaceAll(raw, fmt.Sprintf(".%s", suffix), "")
		if strings.Contains(root, ".") {
			parts := strings.Split(root, ".")
			root = parts[len(parts)-1]
			raw = fmt.Sprintf("%s.%s", root, suffix)
		}
	}

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
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:443", address)
	}
	conn, err := tls.Dial("tcp", address, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return fmt.Sprintf("%v", conn.ConnectionState().PeerCertificates[0].Subject), nil
}
