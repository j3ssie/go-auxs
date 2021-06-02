package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	onlyAscii     bool
	stringCount   int
	stringToCount string
	concurrency   int
	limit         int
)

func main() {
	flag.IntVar(&concurrency, "c", 50, "Set the concurrency level")
	flag.IntVar(&limit, "l", 20, "String length limit")
	flag.StringVar(&stringToCount, "s", "", "String length limit")
	flag.IntVar(&stringCount, "sc", 1, "String length limit")

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
	if len(line) > limit {
		return
	}

	if stringToCount != "" {
		count := strings.Count(line, stringToCount)
		if count > stringCount {
			return
		}
	}

	fmt.Println(line)
}
