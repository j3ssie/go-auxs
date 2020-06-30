package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/launcher"
	"os"
	"strings"
	"sync"
	"time"
)

// Open URL with chromium with your browser session
var (
	verbose     bool
	devTool     bool
	concurrency int
	timeout     int
	proxy       string
	viaBurp     bool
)

func main() {
	// cli aguments
	flag.StringVar(&proxy, "proxy", "", "Proxy")
	flag.BoolVar(&devTool, "dev", false, "Enable Devtools")
	flag.BoolVar(&verbose, "v", false, "Enable popup")
	flag.BoolVar(&viaBurp, "b", false, "Shortcut for -proxy 'http://127.0.0.1' ")
	flag.IntVar(&concurrency, "c", 3, "concurrency ")
	flag.IntVar(&timeout, "t", 3, "minutes to close")
	// custom help
	flag.Usage = func() {
		usage()
		os.Exit(1)
	}
	flag.Parse()
	if viaBurp {
		proxy = "http://127.0.0.1:8080"
	}

	var wg sync.WaitGroup
	// init headless
	base := launcher.New().
		Headless(false). // run chrome on foreground, you can also use env "rod=show"
		Set("proxy-server", proxy).
		// add a flag, here we set a http proxy
		//Devtools(true). // open devtools for each new tab
		Launch()


	browser := rod.New().
		ControlURL(base).
		Trace(true). // show trace of each input action
		Slowmotion(2 * time.Second). // each input action will take 2 second
		Connect().
		Timeout(time.Minute)

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		url := strings.TrimSpace(sc.Text())
		browser.Page(url)
	}
	wg.Wait()

	fmt.Printf("Press 'Ctrl + C' to close or wait for %v minutes\n", timeout)
	time.Sleep(time.Duration(timeout) * time.Minute)
	browser.Close()
}

func usage() {
	func() {
		h := "Open in Chrome \n\n"
		h += "Usage:\n"
		h += "cat list_urls.txt | oic -v \n"
		h += "oic example.com github.com\n"
		fmt.Fprint(os.Stderr, h)
	}()
}
