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
// cat /tmp/list_of_IP | jjoin -nn -j ' '

var (
	delimiterString string
	joinString      string
	data            []string
	newLine         bool
	joinNewline     bool
)

func main() {
	// cli arguments
	flag.StringVar(&delimiterString, "d", ",", "delimiter char to split")
	flag.StringVar(&joinString, "j", " ", "String to join after split")
	flag.BoolVar(&newLine, "n", false, "delimiter char")
	flag.BoolVar(&joinNewline, "nn", false, "delimiter by new line")
	flag.Parse()

	if newLine {
		joinString = "\n"
	}
	if joinString == "nN" {
		joinString = "\n"
	}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if err := sc.Err(); err == nil && line != "" {
			if joinNewline {
				data = append(data, line)
				continue
			}
			handleString(line)
		}
	}

	if joinNewline {
		fmt.Println(strings.Join(data, joinString))
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
