package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"sort"
	"strings"
	"sync"
)

// Examples
// echo 'domain.com' | strr -t '{}.{{.Raw}}' -I wordlists.txt
// echo 'www-{}.domain.com' | strr -I wordlists.txt
// cat domains.txt | strr -t '{}.{{.Raw}}' -i 'dev'

var (
	verbose        bool
	alexa          bool
	extra          bool
	input          string
	inputList      string
	replaceString  string
	templateString string
)

func main() {
	// cli arguments
	var inputs []string
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.StringVar(&inputList, "I", "", "inputList")
	flag.StringVar(&input, "i", "", "inputList")
	flag.StringVar(&replaceString, "s", "{}", "replaceString")
	flag.StringVar(&templateString, "t", "", "templateString")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		args := os.Args[1:]
		sort.Strings(args)
		target := args[len(args)-1]
		if FileExists(target) {
			inputs = append(inputs, ReadingLines(target)...)
		} else {
			inputs = append(inputs, target)
		}
	}
	if input != "" {
		inputs = append(inputs, input)
	}
	if inputList != "" {
		inputs = append(inputs, ReadingLines(inputList)...)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				for _, input := range inputs {
					doReplace(job, input)
				}
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			url := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && url != "" {
				jobs <- url
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func doReplace(raw string, replace string) {
	if templateString != "" {
		new := strings.ReplaceAll(templateString, "{{.Raw}}", raw)
		new = strings.ReplaceAll(new, replaceString, replace)
		fmt.Println(new)
		return
	}

	if strings.Contains(raw, replaceString) {
		new := strings.ReplaceAll(raw, replaceString, replace)
		fmt.Println(new)
		return
	}

	fmt.Printf("%s%s\n", raw, replace)
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
