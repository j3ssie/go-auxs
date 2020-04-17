package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Get alexa rank of list of urls
// Usage: echo '1.2.3.4/24' | arank -c 50
var concurrency int

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 30, "concurrency")

	// custom help
	flag.Usage = func() {
		os.Exit(1)
	}
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for job := range jobs {
			rank, _ := getAlexaRank(job)
			fmt.Printf("%v,%v\n", job, rank)
		}
	}()

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

func getAlexaRank(url string) (string, error) {
	rank := "-1"

	resp, err := http.Get("http://data.alexa.com/data?cli=10&dat=snbamz&url=" + url)
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
