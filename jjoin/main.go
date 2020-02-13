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
// cat /tmp/list_of_IP | jjoin -d " "
// cat /tmp/list_of_IP | jjoin -s '"'
var verbose bool

func main() {
	// cli aguments
	strip := flag.String("s", "", "strip char")
	seperate := flag.String("d", ",", "delimiter char")
	flag.Parse()
	var result []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if *strip != "" {
			line = strings.Replace(line, *strip, "", -1)
		}
		result = append(result, strings.TrimSpace(line))
	}
	fmt.Println(strings.Join(result, *seperate))
}
