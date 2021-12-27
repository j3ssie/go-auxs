package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cornelk/hashmap"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants"
)

type Raw struct {
	Url        string `json:"url"`
	ArchiveUrl string `json:"archive_url"`
	Status     string `json:"status"`
	Mime       string `json:"mime"`
	Hash       string `json:"hash"`
	Timestamp  string `json:"time"`
	Length     string `json:"length"`
}

var (
	IncludeSubs   bool
	PageCheck     bool
	RawOutput     bool
	Verbose       bool
	GetDomainOnly bool
	FilterFlags   string
	OutputSet     = &hashmap.HashMap{}
)

var client = &http.Client{
	Timeout: 5 * time.Minute, // Some sources need long time to query
}

func main() {
	var domains []string
	flag.BoolVar(&IncludeSubs, "subs", false, "include subdomains of target domain")
	flag.BoolVar(&PageCheck, "p", false, "if the data is large, get by pages")
	flag.BoolVar(&RawOutput, "r", false, "print raw output (JSON format)")
	flag.BoolVar(&GetDomainOnly, "a", false, "print domain only")
	flag.BoolVar(&Verbose, "v", false, "enable verbose")
	flag.StringVar(&FilterFlags, "f", "", "Wayback Machine filter (filter=statuscode:200&filter=!mimetype:text/html)")
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

type fetch func(string) error

func Run(domain string) {
	fetchers := []fetch{getWaybackUrls, getCommonCrawlURLs, getOtxUrls}
	for _, fn := range fetchers {
		err := fn(domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			continue
		}

	}
}

type OTXResult struct {
	URLList []struct {
		URL      string `json:"url"`
		Date     string `json:"date"`
		Domain   string `json:"domain"`
		Hostname string `json:"hostname"`
		Result   struct {
			Urlworker struct {
				IP       string `json:"ip"`
				HTTPCode int    `json:"http_code"`
			} `json:"urlworker"`
			Safebrowsing struct {
				Matches []interface{} `json:"matches"`
			} `json:"safebrowsing"`
		} `json:"result"`
		Httpcode int           `json:"httpcode"`
		Gsb      []interface{} `json:"gsb"`
		Encoded  string        `json:"encoded"`
	} `json:"url_list"`
	PageNum    int  `json:"page_num"`
	Limit      int  `json:"limit"`
	Paged      bool `json:"paged"`
	HasNext    bool `json:"has_next"`
	FullSize   int  `json:"full_size"`
	ActualSize int  `json:"actual_size"`
}

func getOtxUrls(hostname string) error {
	page := 0
	for {
		currentURL := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=50&page=%d", hostname, page)

		r, err := client.Get(currentURL)
		if err != nil {
			return errors.New(fmt.Sprintf("http request to OTX failed: %s", err.Error()))
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return nil
		}
		if Verbose {
			printStderr(currentURL)
		}
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return errors.New(fmt.Sprintf("error reading body from alienvault: %s", err.Error()))
		}
		results := &OTXResult{}
		err = jsoniter.Unmarshal(bytes, results)
		if err != nil {
			return errors.New(fmt.Sprintf("could not decode json response from alienvault: %s", err.Error()))
		}
		for _, result := range results.URLList {
			if result.URL == "" {
				continue
			}

			if RawOutput {
				o := Raw{Url: result.URL, Status: strconv.Itoa(result.Httpcode)}
				if r, err := jsoniter.MarshalToString(o); err == nil {
					fmt.Println(r)
				}
			} else {
				if GetDomainOnly {
					if u, err := url.Parse(result.URL); err == nil {
						hostname := u.Hostname()
						if notExist := OutputSet.Insert(hostname, true); notExist {
							fmt.Println(hostname)
						}
					}
				} else {
					fmt.Println(result.URL)
				}
			}
		}
		if !results.HasNext {
			break
		}
		page++
	}
	return nil
}

func getWaybackUrls(hostname string) error {
	wildcard := "*."
	if !IncludeSubs {
		wildcard = ""
	}

	// Remove `collapse` because we need to get as much as possible
	baseURL := fmt.Sprintf("http://web.archive.org/cdx/search/cdx?url=%s%s/*&output=json", wildcard, hostname)
	if FilterFlags != "" {
		baseURL += "&" + FilterFlags
	}
	printStderr("Base URL: " + baseURL)
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
				if Verbose {
					printStderr(pageDataURL)
				}
				err := downloadWaybackResults(pageDataURL)
				if err != nil {
					printStderr(fmt.Sprintf("Failed to download url %s: %s", pageDataURL, err))
					return
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
		return nil
	} else {
		err := downloadWaybackResults(baseURL)
		if err != nil {
			return err
		}
		return nil
	}
}

func downloadWaybackResults(downloadURL string) error {
	retryTimes := 20
	for retryTimes > 0 {
		if retryTimes < 20 {
			printStderr(fmt.Sprintf("%s: retry %d", downloadURL, 60-retryTimes))
		}
		time.Sleep(time.Second * 5)
		resp, err := client.Get(downloadURL)
		if err != nil {
			retryTimes--
			continue
		}

		defer resp.Body.Close()
		switch resp.StatusCode {
		case 200:
			dec := jsoniter.NewDecoder(resp.Body)
			for dec.More() {
				first := true
				var results [][]string
				if err := dec.Decode(&results); err == nil {
					for _, result := range results {
						if first {
							// skip first result from wayback machine
							// always is "original"
							first = false
							continue
						}
						if result[3] != "" && result[3] == "warc/revisit" {
							continue
						}
						if result[2] == "" {
							continue
						}
						if GetDomainOnly {
							if u, err := url.Parse(result[2]); err == nil {
								hostname := u.Hostname()
								if notExist := OutputSet.Insert(hostname, true); notExist {
									fmt.Println(hostname)
								}
							}
							continue
						}

						if RawOutput {
							o := Raw{
								Timestamp: result[1],
								Url:       result[2],
								Mime:      result[3],
								Status:    result[4],
								Hash:      result[5],
								Length:    result[6],
							}
							o.ArchiveUrl = fmt.Sprintf("https://web.archive.org/web/%sid_/%s", o.Timestamp, o.Url)
							if r, err := jsoniter.MarshalToString(o); err == nil {
								fmt.Println(r)
							}
						} else {
							fmt.Println(result[2])
						}
					}
				}
			}
			return nil
		case 400:
			return errors.New("Status code: 400")
		case 429:
			retryTimes--
		default:
			retryTimes--
		}
	}
	return errors.New("Max try times")
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

func getCommonCrawlURLs(domain string) error {
	wildcard := "*."
	if !IncludeSubs {
		wildcard = ""
	}
	currentApis, err := getCurrentCC()
	if err != nil {
		return fmt.Errorf("error getting current commoncrawl url: %v", err)
	}
	var wg sync.WaitGroup
	pool, _ := ants.NewPoolWithFunc(5, func(i interface{}) {
		defer wg.Done()
		currentApi := i.(string)
		currentURL := fmt.Sprintf("%s?url=%s%s/*&output=json", currentApi, wildcard, domain)
		res, err := http.Get(currentURL)
		if err != nil {
			return
		}

		if Verbose {
			printStderr(currentURL)
		}
		defer res.Body.Close()

		sc := bufio.NewScanner(res.Body)

		for sc.Scan() {
			result := struct {
				Urlkey       string `json:"urlkey"`
				Timestamp    string `json:"timestamp"`
				URL          string `json:"url"`
				Mime         string `json:"mime"`
				MimeDetected string `json:"mime-detected"`
				Status       string `json:"status"`
				Digest       string `json:"digest"`
				Length       string `json:"length"`
				Offset       string `json:"offset"`
				Filename     string `json:"filename"`
				Languages    string `json:"languages"`
				Encoding     string `json:"encoding"`
			}{}
			err = jsoniter.Unmarshal([]byte(sc.Text()), &result)

			if err != nil {
				continue
			}
			if result.URL == "" {
				continue
			}
			// ! We only need to unique in this case
			if GetDomainOnly {
				if u, err := url.Parse(result.URL); err == nil {
					hostname := u.Hostname()
					if notExist := OutputSet.Insert(hostname, true); notExist {
						fmt.Println(hostname)
					}
				}
				continue
			}

			if RawOutput {
				o := Raw{
					Timestamp: result.Timestamp,
					Url:       result.URL,
					Mime:      result.Mime,
					Status:    result.Status,
					Hash:      result.Digest,
					Length:    result.Length,
				}
				if r, err := jsoniter.MarshalToString(o); err == nil {
					fmt.Println(r)
				}
			} else {
				fmt.Println(result.URL)

			}
		}
	})
	defer pool.Release()
	for _, currentApi := range currentApis {
		printStderr(fmt.Sprintf("Downloading page: %s", currentApi))
		wg.Add(1)
		pool.Invoke(currentApi)
	}
	wg.Wait()

	return nil
}

type CommonCrawlInfo []struct {
	API string `json:"cdx-api"`
}

func getCurrentCC() ([]string, error) {
	r, err := client.Get("http://index.commoncrawl.org/collinfo.json")
	if err != nil {
		return []string{}, err
	}
	defer r.Body.Close()
	resp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return []string{}, err
	}
	var wrapper CommonCrawlInfo
	err = jsoniter.Unmarshal(resp, &wrapper)
	if err != nil {
		return []string{}, fmt.Errorf("could not unmarshal json from CC: %s", err.Error())
	}
	if len(wrapper) < 1 {
		return []string{}, errors.New("unexpected response from commoncrawl.")
	}
	var CCes []string
	for _, CC := range wrapper {
		CCes = append(CCes, CC.API)
	}
	return CCes, nil
}

func printStderr(msg string) {
	fmt.Fprintf(os.Stderr, msg+"\n")
}
