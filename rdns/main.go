package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
)

// Extract domain from IP Address info via reverse DNS
// cat /tmp/list_of_IP | rdns -c 100
var (
	verbose  bool
	alexa    bool
	resolver string
	proto    string
)

func main() {
	// cli aguments
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&alexa, "a", false, "Check Alexa Rank of domain")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.StringVar(&resolver, "s", "8.8.8.8:53", "Resolver")
	flag.StringVar(&proto, "p", "tcp", "protocol to do reverse DNS")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Strings(args)
		raw := args[len(args)-1]
		getDomain(raw)
		os.Exit(0)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				getDomain(job)
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

func getDomain(raw string) {
	r := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, proto, resolver)
		},
	}

	domains, err := r.LookupAddr(context.Background(), raw)
	if err != nil {
		return
	}
	for _, d := range domains {
		domain := strings.TrimRight(d, ".")
		if !alexa {
			fmt.Printf("%s,%s\n", raw, domain)
			continue
		}
		rank, _ := getAlexaRank(domain)
		fmt.Printf("%s,%s,%s\n", raw, domain, rank)
	}
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
