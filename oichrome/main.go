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

	"github.com/chromedp/chromedp"
)

// Open URL with chromium with your browser session
var verbose bool
var concurrency int

func main() {
	// cli aguments
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.IntVar(&concurrency, "c", 3, "concurrency ")
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

func openWithChrome(url string) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", false),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	// create context
	ctx, _ := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	// defer cancel()

	// navigate to a page, wait for an element, click
	// var example string
	var res string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		// wait for footer element is visible (ie, page is loaded)
		chromedp.WaitVisible(`div > footer`),
	)
	fmt.Println(res)
	if err != nil {
		fmt.Println("something wrong")
		log.Fatal(err)
	}
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
