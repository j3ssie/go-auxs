package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/c-robinson/iplib"
)

// Extend the IP range by CIDR
// Usage: echo '1.2.3.4/24' | eip -s 32
var concurrency int
var sub int

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 3, "concurrency ")
	flag.IntVar(&sub, "s", 32, "CIDR subnet (e.g: 24, 22)")

	// custom help
	flag.Usage = func() {
		os.Exit(1)
	}
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	//for i := 0; i < concurrency; i++ {
	//	wg.Add(1)
	//	go func() {
	//		defer wg.Done()
	//		for job := range jobs {
	//			openWithChrome(job)
	//		}
	//	}()
	//}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for job := range jobs {
			extendRange(job, sub)
		}
	}()

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

func extendRange(rangeIP string, sub int) {
	_, ipna, err := iplib.ParseCIDR(rangeIP)
	if err != nil {
		return
	}
	extendedIPs, err := ipna.Subnet(sub)
	if err != nil {
		return
	}
	for _, item := range extendedIPs {
		ip := item.String()
		if sub == 32 {
			ip = item.IP.String()
		}
		fmt.Println(ip)
	}

}
