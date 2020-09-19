package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants"
)

type OTXResult struct {
	HasNext bool `json:"has_next"`
	URLList []struct {
		URL string `json:"url"`
	} `json:"url_list"`
}

type CommonCrawlInfo []struct {
	API string `json:"cdx-api"`
}

var IncludeSubs bool
var PageCheck bool
var client = &http.Client{
	Timeout: 2 * time.Minute, // Some sources need long time to query
}

func main() {
	var domains []string
	flag.BoolVar(&IncludeSubs, "subs", false, "include subdomains of target domain")
	flag.BoolVar(&PageCheck, "p", false, "if the data is large, get by pages")
	flag.Parse()
	if flag.NArg() > 0 {
		domains = []string{flag.Arg(0)}
	} else {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line != "" {
				domains = append(domains, line)
			}
		}
	}
	for _, domain := range domains {
		Run(domain)
	}
}

type fetch func(string) ([]string, error)

func Run(domain string) {
	fetchers := []fetch{getWaybackUrls, getCommonCrawlURLs, getOtxUrls}
	for _, fn := range fetchers {
		found, err := fn(domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			continue
		}
		for _, f := range found {
			fmt.Println(f)
		}
	}
}

func getOtxUrls(hostname string) ([]string, error) {
	var urls []string
	page := 0
	for {
		r, err := client.Get(fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/hostname/%s/url_list?limit=50&page=%d", hostname, page))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("http request to OTX failed: %s", err.Error()))
		}
		defer r.Body.Close()
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error reading body from alienvault: %s", err.Error()))
		}
		o := &OTXResult{}
		err = jsoniter.Unmarshal(bytes, o)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("could not decode json response from alienvault: %s", err.Error()))
		}
		for _, url := range o.URLList {
			urls = append(urls, url.URL)
		}
		if !o.HasNext {
			break
		}
		page++
	}
	return urls, nil
}
func getWaybackUrls(hostname string) ([]string, error) {
	wildcard := "*."
	if !IncludeSubs {
		wildcard = ""
	}
	baseURL := fmt.Sprintf("http://web.archive.org/cdx/search/cdx?url=%s%s/*&output=json&collapse=urlkey&fl=original", wildcard, hostname)
	if PageCheck {
		// Check amount of page first
		pagesNumURL := baseURL + "&showNumPages=true"
		pagesNum := getWaybackPages(pagesNumURL)
		if pagesNum > 0 {
			printStderr(fmt.Sprintf("Total pages: %d", pagesNum))

			var wg sync.WaitGroup
			pool, _ := ants.NewPoolWithFunc(15, func(i interface{}) {
				defer wg.Done()
				page := i.(int)
				pageDataURL := fmt.Sprintf("%s&page=%d", baseURL, page)
				results, err := downloadWaybackResults(pageDataURL)
				if err != nil {
					printStderr(fmt.Sprintf("Failed to download url %s: %s", pageDataURL, err))
					return
				}
				for _, result := range results {
					fmt.Println(result)
				}
			})
			defer pool.Release()
			for i := 0; i <= pagesNum; i++ {
				printStderr(fmt.Sprintf("Downloading page: %d/%d", i, pagesNum))
				wg.Add(1)
				pool.Invoke(i)
			}
			wg.Wait()
		}
		return []string{}, nil
	} else {
		results, err := downloadWaybackResults(baseURL)
		if err != nil {
			return []string{}, err
		}
		return results, nil
	}

}

func downloadWaybackResults(url string) ([]string, error) {
	var resp *http.Response
	retryTimes := 60
retry:
	for retryTimes > 0 {
		if retryTimes < 60 {
			printStderr(fmt.Sprintf("%s: retry %d", url, 60-retryTimes))
		}
		time.Sleep(time.Second * 5)
		r, err := client.Get(url)
		if err != nil {
			retryTimes--
			continue
		}
		switch r.StatusCode {
		case 200:
			resp = r
			break retry
		case 400:
			return []string{}, nil
		case 429:
			retryTimes--
		default:
			retryTimes--
		}
	}
	if resp == nil {
		return nil, errors.New(fmt.Sprintf("Body empty"))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error reading body: %s", err.Error()))
	}
	resp.Body.Close()

	var waybackresp [][]string
	var found []string
	err = jsoniter.Unmarshal(body, &waybackresp)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("could not decoding response from wayback machine: %s", err.Error()))
	}
	first := true
	for _, result := range waybackresp {
		if first {
			// skip first result from wayback machine
			// always is "original"
			first = false
			continue
		}
		found = append(found, result[0])
	}
	return found, nil
}

func getWaybackPages(url string) int {
	r, err := client.Get(url)
	if err != nil {
		printStderr("Request WayBack Pages error")
		return -1
	}
	defer r.Body.Close()
	resp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		printStderr("Read WayBack body error")
		return -1
	}
	body := strings.TrimSpace(string(resp))
	amount, err := strconv.Atoi(body)
	if err != nil {
		printStderr("Convert body to int error")
		return -1
	}
	return amount
}

func getCommonCrawlURLs(domain string) ([]string, error) {
	var found []string
	wildcard := "*."
	if !IncludeSubs {
		wildcard = ""
	}
	currentApi, err := getCurrentCC()
	if err != nil {
		return nil, fmt.Errorf("error getting current commoncrawl url: %v", err)
	}
	res, err := http.Get(
		fmt.Sprintf("%s?url=%s%s/*&output=json", currentApi, wildcard, domain),
	)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	sc := bufio.NewScanner(res.Body)

	for sc.Scan() {
		wrapper := struct {
			URL string `json:"url"`
		}{}
		err = jsoniter.Unmarshal([]byte(sc.Text()), &wrapper)

		if err != nil {
			continue
		}

		found = append(found, wrapper.URL)
	}
	return found, nil
}

func getCurrentCC() (string, error) {
	r, err := client.Get("http://index.commoncrawl.org/collinfo.json")
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	resp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	var wrapper CommonCrawlInfo
	err = jsoniter.Unmarshal(resp, &wrapper)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal json from CC: %s", err.Error())
	}
	if len(wrapper) < 1 {
		return "", errors.New("unexpected response from commoncrawl.")
	}
	return wrapper[0].API, nil
}

func printStderr(msg string) {
	fmt.Fprintf(os.Stderr, msg+"\n")
}
