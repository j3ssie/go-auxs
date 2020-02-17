package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Open URL with chromium with your browser session
var verbose bool
var concurrency int
var timeout int

func main() {
	// cli aguments
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.IntVar(&concurrency, "c", 3, "concurrency ")
	flag.IntVar(&timeout, "t", 10, "timeout ")
	// custom help
	flag.Usage = func() {
		usage()
		os.Exit(1)
	}
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				openWithChrome(job)
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

// SendWithChrome send request with real browser
func openWithChrome(url string) error {
	// show the chrome page in debug mode
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", false),
		chromedp.Flag("disable-setuid-sandbox", true),
	)

	allocCtx, bcancel := chromedp.NewExecAllocator(context.Background(), opts...)
	allocCtx, bcancel = context.WithTimeout(allocCtx, time.Duration(timeout)*time.Second)
	defer bcancel()
	chromeContext, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()
	// start Chrome and run given tasks
	err := chromedp.Run(
		chromeContext,
		chromedp.Navigate(url),
		// wait for the page to load
		chromedp.Sleep(time.Duration(timeout)*time.Second),

	)
	if err != nil {
		return err
	}

	return nil
}

func usage() {
	func() {
		h := "Open in Chrome \n\n"
		h += "Usage:\n"
		h += "cat list_urls.txt | oichrome -v \n"
		h += "oichrome example.com github.com\n"
		fmt.Fprint(os.Stderr, h)
	}()
}
