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
var verbose bool

func main() {
	// cli aguments
	seperate := flag.String("d", ",", "delimiter char")
	flag.Parse()
	var result []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		result = append(result, strings.TrimSpace(sc.Text()))
	}
	fmt.Println(strings.Join(result, *seperate))
}
