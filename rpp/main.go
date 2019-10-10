package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Gen string from list with pattern
func main() {
	var patterns []string
	patterns = os.Args[1:]
	if patterns == nil {
		usage()
		os.Exit(1)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// recive list of url from stdin
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			line := sc.Text()
			for _, pattern := range patterns {
				fmt.Println(strings.Replace(pattern, "[i]", line, -1))
			}
		}
	}
}

// usage
func usage() {
	func() {
		h := "Replace with pattern \n\n"
		h += "Usage:\n"
		h += `cat whatever.txt | rpp https://google.com/?q=site%3Agithub.com%20[i]`
		h += "\n"
		fmt.Fprint(os.Stderr, h)
	}()
}
