package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// Only get IP or CIDR from list of input
func main() {
	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		domain := sc.Text()

		func() {
			// var val string
			if resolved, err := net.LookupHost(domain); err == nil {
				fmt.Printf("%v,%v \n", domain, resolved[0])
			}

		}()
	}

}
