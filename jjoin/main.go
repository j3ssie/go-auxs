package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// just join all the content with character
// cat /tmp/list_of_IP | jjoin
// cat /tmp/list_of_IP | jjoin -d "."
// cat /tmp/list_of_IP | jjoin -j ' '
var (
	delimiterString string
	joinString      string
	newLine         bool
)

func main() {
	// cli aguments
	flag.StringVar(&delimiterString, "d", ",", "delimiter char to split")
	flag.StringVar(&joinString, "j", " ", "String to join after split")
	flag.BoolVar(&newLine, "n", false, "delimiter char")
	flag.Parse()

	if newLine {
		joinString = "\n"
	}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if err := sc.Err(); err == nil && line != "" {
			handleString(line)
		}
	}
}

func handleString(raw string) {
	var result []string
	if !strings.Contains(raw, delimiterString) {
		result = append(result, raw)
		fmt.Println(strings.Join(result, joinString))
		return
	}

	result = strings.Split(raw, delimiterString)
	fmt.Println(strings.Join(result, joinString))
}
