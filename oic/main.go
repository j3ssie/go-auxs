package main

import (
	"bufio"
	"flag"
	"github.com/skratchdot/open-golang/open"
	"os"
	"sort"
	"strings"
)

// Open URL with your default browser
// Usage:
// cat urls.txt | oic
// cat urls.txt | oic -a 'Google Chrome Canary'
// oic http://example.com

var (
	verbose     bool
	concurrency int
	appName     string
	data        string
	dataFile    string
)

func main() {
	// cli args
	flag.StringVar(&appName, "a", "", "App name")
	flag.StringVar(&data, "u", "", "URL to open")
	flag.StringVar(&dataFile, "U", "", "URL to open")
	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.IntVar(&concurrency, "c", 5, "number of tab at a time")
	flag.Parse()

	// get app name from ENV
	if appName == "" {
		newApp, ok := os.LookupEnv("OIC_APP")
		if ok {
			appName = newApp
		}
	}

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
		OpenString(raw)
		os.Exit(0)
	}

	var count int
	for _, raw := range inputs {
		if count == concurrency {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
		}
		OpenString(raw)
		count++
	}
}

func OpenString(raw string) {
	if appName != "" {
		open.RunWith(raw, appName)
		return
	}
	open.Run(raw)
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
