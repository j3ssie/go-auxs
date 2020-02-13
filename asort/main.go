package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Sort list of domain but by alexa rank
// cat /tmp/list_of_domain | asort
var (
	verbose bool
)

func main() {
	// cli aguments
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	data := make(map[int]string)
	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				hostname := getHostName(job, "")
				if hostname != "" {
					rank, err := getAlexaRank(hostname)
					if err != nil {
						continue
					}
					_, exist := data[rank]
					if exist {
						data[rank] += ";" + hostname
					} else {
						data[rank] = hostname
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
	doSort(data)
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

func doSort(data map[int]string) {
	// To store the keys in slice in sorted order
	var keys []int
	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		if !strings.Contains(data[k], ";") {
			if verbose {
				fmt.Printf("%v,%v\n", data[k], k)
			} else {
				fmt.Println(data[k])
			}
			continue
		}
		var hosts []string
		hosts = strings.Split(data[k], ";")
		if len(hosts) > 0 {
			for _, host := range hosts {
				if verbose {
					fmt.Printf("%v,%v\n", host, k)
				} else {
					fmt.Println(host)
				}
			}
		}
	}

}

func getAlexaRank(url string) (int, error) {
	rank := "-1"

	resp, err := http.Get("http://data.alexa.com/data?cli=10&dat=snbamz&url=" + url)
	if err != nil {
		return -1, err
	}

	defer resp.Body.Close()

	alexaData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
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

	i, err := strconv.Atoi(rank)
	if err != nil {
		return 988888888888888, nil
	}
	return i, nil
}
