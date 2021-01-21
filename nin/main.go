package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/thoas/go-funk"
)

// Exclude file from a file
// cat file.txt | nin -e excldue.txt
var (
	verbose bool
	alexa   bool
	extra   bool
	exclude string
)
var logger *log.Logger

func main() {
	// cli aguments
	flag.StringVar(&exclude, "e", "", "Exclude File")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	args := os.Args[1:]
	sort.Strings(args)
	exclude = args[len(args)-1]

	if !FileExists(exclude) {
		log.Printf("No input found: %s", exclude)
		os.Exit(-1)
	}

	var stdInputs, excludeInputs []string
	log.Printf("No input found: %s", exclude)
	excludeInputs = ReadingLines(exclude)

	// detect if anything came from std
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			target := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && target != "" {
				stdInputs = append(stdInputs, target)
			}
		}
	}

	log.Printf("Excludes inputs length: %v", len(excludeInputs))
	log.Printf("Inputs length: %v", len(stdInputs))

	// really do something
	for _, input := range stdInputs {
		//for _, einput := range excludeInputs {
		if !funk.ContainsString(excludeInputs, input) {
			fmt.Println(input)
		}
	}

}

// FileExists check if file is exist or not
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// ReadingLines Reading file and return content as []string
func ReadingLines(filename string) []string {
	var result []string
	if strings.HasPrefix(filename, "~") {
		filename, _ = homedir.Expand(filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := strings.TrimSpace(scanner.Text())
		if val == "" {
			continue
		}
		result = append(result, val)
	}

	if err := scanner.Err(); err != nil {
		return result
	}
	return result
}
