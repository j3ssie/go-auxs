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
var verbose bool

func main() {
	// cli aguments
	var seperate string
	flag.StringVar(&seperate, "s", ",", "seperate string")

	var result []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		result = append(result, strings.TrimSpace(sc.Text()))
	}
	fmt.Println(strings.Join(result, seperate))
}
