package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Open URL with chromium with your browser session

var verbose bool
var chromePath string

func main() {
	// cli aguments
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	chromePath = getChrome()
	// custom help
	flag.Usage = func() {
		usage()
		os.Exit(1)
	}
	flag.Parse()

	// open from stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// recive list of url from stdin
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url := sc.Text()
			open(url)
		}
		os.Exit(0)
	}

	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}
	var args []string
	rawArgs := os.Args[1:]
	sort.Sort(sort.StringSlice(rawArgs))
	args = rawArgs
	// only get url target
	if rawArgs[0] == "-v" {
		args = rawArgs[1:]
	}
	if rawArgs[0] == "-c" {
		args = rawArgs[3:]
	}

	// fmt.Printf("raw argv: %v\n", rawArgs)
	// fmt.Printf("argv: %v\n", args)
	for _, url := range args {
		open(url)
	}
}

func open(url string) {
	if verbose {
		fmt.Printf("Open with chrome: %v \n", url)
	}
	cmd := fmt.Sprintf("%v %v", chromePath, url)
	run(cmd)
}

func run(realCmd string) {
	var cmd *exec.Cmd
	command := strings.Split(realCmd, ` `)
	cmd = exec.Command(command[0], command[1:]...)
	// cmd.Run()
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func getChrome() string {
	var chromePath string
	// Common paths for Google Chrome or chromium
	paths := []string{
		"/snap/bin/chromium",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		chromePath = path
		break
	}
	return chromePath
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
