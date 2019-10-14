package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

func main() {
	// resolve bunch of domains to IP
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	var getDomain bool
	flag.BoolVar(&getDomain, "d", false, "Verbose output")
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)
	output := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// really resolved it here
				if resolved, err := net.LookupHost(job); err == nil {
					if getDomain {
						record := job + "," + resolved[0]
						output <- record
					} else {
						output <- resolved[0]

					}
				}
			}
		}()
	}

	seen := make(map[string]bool)
	go func() {
		for record := range output {
			if _, ok := seen[record]; !ok {
				seen[record] = true
				fmt.Println(record)
			} else {
				continue
			}
		}
		close(output)
	}()

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			domain := strings.TrimSpace(sc.Text())
			jobs <- domain
		}
		close(jobs)
	}()

	wg.Wait()

}
