package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Open URL with your default browser
// Usage:
// cat urls.txt | oic
// cat urls.txt | oic -c 5 -proxy http://127.0.0.1:8080

var (
	verbose     bool
	headless    bool
	concurrency int
	timeout     int
	data        string
	dataFile    string
	proxy       string
)

func main() {
	// cli args
	flag.StringVar(&data, "u", "", "URL to open")
	flag.StringVar(&dataFile, "U", "", "URL to open")
	flag.IntVar(&concurrency, "c", 5, "number of tab at a time")
	flag.IntVar(&timeout, "t", 15, "timeout in second")
	flag.StringVar(&proxy, "proxy", "", "proxy to pass chrome to (eg: http://127.0.0.1:8080)")
	flag.BoolVar(&headless, "q", false, "enable headless")
	flag.Parse()

	// detect if anything came from std
	var inputs []string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			target := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && target != "" {
				inputs = append(inputs, target)
			}
		}
	}

	if data != "" {
		inputs = append(inputs, data)
	}
	if dataFile != "" {
		inputs = append(inputs, ReadingLines(dataFile)...)
	}

	if (stat.Mode()&os.ModeCharDevice) != 0 && len(inputs) == 0 {
		args := os.Args[1:]
		sort.Strings(args)
		raw := args[len(args)-1]
		RequestWithChrome(raw)
		os.Exit(0)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				RequestWithChrome(job)
			}
		}()
	}

	go func() {
		for _, raw := range inputs {
			jobs <- raw
		}
		close(jobs)
	}()
	wg.Wait()
}

// RequestWithChrome Do request with real browser
func RequestWithChrome(url string) string {
	// prepare the chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		//chromedp.UserDataDir(""),
	)

	if proxy != "" {
		opts = append(opts, chromedp.ProxyServer(proxy))
	}

	allocCtx, bcancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer bcancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// run task list
	var data string
	contentID := "main"
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.OuterHTML(contentID, &data, chromedp.NodeVisible, chromedp.ByID),
	)

	cleanUp()
	if err != nil {
		return ""
	}
	return data
}

func cleanUp() {
	tmpFolder := path.Join(os.TempDir(), "chromedp-runner*")
	if _, err := os.Stat("/tmp/"); !os.IsNotExist(err) {
		tmpFolder = path.Join("/tmp/", "chromedp-runner*")
	}
	junks, err := filepath.Glob(tmpFolder)
	if err != nil {
		return
	}
	for _, junk := range junks {
		os.RemoveAll(junk)
	}
}

// ReadingLines Reading file and return content as []string
func ReadingLines(filename string) []string {
	var result []string
	file, err := os.Open(filename)
	if err != nil {
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := strings.TrimSpace(scanner.Text())
		if val == "" {
			continue
		}
		result = append(result, val)
	}

	if err := scanner.Err(); err != nil {
		return result
	}
	return result
}
