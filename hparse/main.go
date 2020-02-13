package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Finding attr in every HTML tag
// cat /tmp/list_of_IP | hparse -t 'a'
var (
	tag  string
	attr string
)

func main() {
	// cli arguments
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.StringVar(&tag, "t", "a", "Tag name")
	flag.StringVar(&attr, "a", "href", "Attribute name")
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// do something
				result, err := doParse(job, tag, attr)
				if err == nil {
					fmt.Print(result)
				}
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			u := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && u != "" {
				jobs <- u
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func doParse(url string, tag string, attr string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result []string
	var data string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return "", err
	}
	doc.Find(tag).Each(func(i int, s *goquery.Selection) {
		if attr == "text" {
			result = append(result, s.Text())
			return
		}
		href, ok := s.Attr(attr)
		if ok {
			result = append(result, href)
		}
	})

	if len(result) > 0 {
		data = strings.Join(result, "\n")
	}
	return data, nil
}
