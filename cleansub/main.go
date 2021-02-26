package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

var nameStripRE = regexp.MustCompile(`^u[0-9a-f]{4}|20|22|25|2b|2f|3d|3a|40`)
var subdomainRE = regexp.MustCompile(`(([a-zA-Z0-9]{1}|[_a-zA-Z0-9]{1}[_a-zA-Z0-9-]{0,61}[a-zA-Z0-9]{1})[.]{1})+[a-zA-Z]{2,61}`)
var subwithIPv4 = regexp.MustCompile(`(?m)[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.`)
var subwithIPv42 = regexp.MustCompile(`(?m)[0-9]{1,3}\-[0-9]{1,3}\-[0-9]{1,3}`)

var (
	target string

	concurrency int
)

func main() {
	flag.StringVar(&target, "t", "", "Specify target to clean")
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")

	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				checkClean(job)
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			url := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && url != "" {
				jobs <- url
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func checkClean(line string) {
	name := subdomainRE.FindString(line)
	name = strings.ToLower(name)
	for {
		name = strings.Trim(name, "-.")
		if i := nameStripRE.FindStringIndex(name); i != nil {
			name = name[i[1]:]
		} else {
			break
		}
	}
	name = removeAsteriskLabel(name)

	// ignore if
	if target != "" {
		if !strings.Contains(name, target) && (name == target) {
			return
		}
	}

	isWildCard := removeWildcard(name)
	if isWildCard {
		return
	}

	fmt.Println(name)
}

func removeWildcard(s string) bool {
	matched := subwithIPv4.MatchString(s)
	if matched {
		return matched
	}

	matched = subwithIPv42.MatchString(s)
	if matched {
		return matched
	}
	return false
}

func removeAsteriskLabel(s string) string {
	startIndex := strings.LastIndex(s, "*.")

	if startIndex == -1 {
		return s
	}

	return s[startIndex+2:]
}
