package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type Result struct {
	Input            map[string]string `json:"input"`
	Position         int               `json:"position"`
	StatusCode       int64             `json:"status"`
	ContentLength    int64             `json:"length"`
	ContentWords     int64             `json:"words"`
	ContentLines     int64             `json:"lines"`
	RedirectLocation string            `json:"redirectlocation"`
	ResultFile       string            `json:"resultfile"`
	Url              string            `json:"url"`
}

type jsonFileOutput struct {
	CommandLine string   `json:"commandline"`
	Time        string   `json:"time"`
	Results     []Result `json:"results"`
}

func main() {
	var limit int
	flag.IntVar(&limit, "l", 100, "Limit")
	flag.Parse()
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file")
		os.Exit(1)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file")
		os.Exit(1)
	}
	foundUrls := &jsonFileOutput{}
	err = jsoniter.Unmarshal(data, foundUrls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to Unmarshal: %s", err)
		os.Exit(1)
	}

	resultsPerHost := make(map[string][]Result)
	for _, result := range foundUrls.Results {
		u, err := url.Parse(result.Url)
		if err != nil {
			continue
		}
		hostname := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

		if rList := resultsPerHost[hostname]; len(rList) == 0 {
			resultsPerHost[hostname] = []Result{result}
		} else {
			resultsPerHost[hostname] = append(resultsPerHost[hostname], result)
		}
	}

	var results []Result
	for _, resultList := range resultsPerHost {
		p := getSpams(resultList)
		if len(p) == 0 {
			continue
		}
		// len(blackList)) == 0 mean get all
		var blackList []string
		for statusCode, times := range p {
			if times >= limit {
				blackList = append(blackList, statusCode)
			}
		}

		for _, found := range resultList {
			if !isSpam(blackList, found) {
				results = append(results, found)
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		siUrl, err := url.Parse(results[i].Url)
		if err != nil {
			return true
		}
		sjUrl, err := url.Parse(results[j].Url)
		if err != nil {
			return true
		}

		siLower := strings.ToLower(siUrl.Hostname())
		sjLower := strings.ToLower(sjUrl.Hostname())
		if siLower < sjLower {
			return true
		} else {
			return false
		}
	})

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].StatusCode < results[j].StatusCode {
			return true
		} else {
			return false
		}
	})

	for _, r := range results {
		f := fmt.Sprintf("%s,%d,%d,%d,%d,%s", r.Url, r.StatusCode, r.ContentLength, r.ContentWords, r.ContentLines, r.RedirectLocation)
		fmt.Println(f)
	}
}

func getSpams(results []Result) map[string]int {
	m := make(map[string]int)
	for _, r := range results {
		f := fmt.Sprintf("%d,%d,%d,%d", r.StatusCode, r.ContentLines, r.ContentWords)
		if _, ok := m[f]; !ok {
			m[f] = 1
		} else {
			m[f]++
		}
	}
	return m
}

func isSpam(blacklist []string, r Result) bool {
	f := fmt.Sprintf("%d,%d,%d,%d", r.StatusCode, r.ContentLines, r.ContentWords)
	for _, e := range blacklist {
		if f == e {
			return true
		}
	}
	return false
}
