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
				// if resolved, err := net.LookupHost(job); err == nil {
				if resolved, err := net.LookupHost(job); err == nil {
					output <- resolved[0]
				}
			}
		}()
	}

	seen := make(map[string]bool)
	go func() {
		for ip := range output {
			if _, ok := seen[ip]; !ok {
				seen[ip] = true
				fmt.Println(ip)
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
