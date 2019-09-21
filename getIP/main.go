package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// Only get IP or CIDR from list of input
func main() {
	seen := make(map[string]bool)
	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		domain := sc.Text()

		func() {
			var val string
			if _, ipv4Net, err := net.ParseCIDR(domain); err == nil {
				val = ipv4Net.String()
			}

			if net.ParseIP(domain) != nil {
				val = domain
			}

			if seen[val] == false && val != "" {
				fmt.Println(val)
			}
			seen[val] = true
		}()
	}

}
