package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Cat Interesting

type Options struct {
	file      string
	column    string
	delimiter string
	threshold float64
	verbose   bool
}

func main() {
	// cli aguments
	opts := Options{}
	flag.StringVar(&opts.file, "f", "", "file to list")
	flag.StringVar(&opts.column, "c", "-1", "columns to focus, default entire line e.g: -c 0,3")
	flag.StringVar(&opts.delimiter, "d", ",", "delimiter character")
	flag.Float64Var(&opts.threshold, "t", 0.95, "threshold")
	flag.BoolVar(&opts.verbose, "v", false, "verbose")
	flag.Parse()

	// reading from stdin
	var data []string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// recive list of url from stdin
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			data = append(data, sc.Text())
		}
	}

	// reading from argument
	if opts.file != "" {
		text, err := os.Open(opts.file)
		if err != nil {
			log.Fatal(err)
		}
		defer text.Close()
		scc := bufio.NewScanner(text)
		for scc.Scan() {
			data = append(data, scc.Text())
		}
	}

	// fmt.Println(data)
	interesting(data, &opts)

}

func interesting(data []string, opt *Options) {
	var ignoreCols []int

	// get ignore column
	if len(opt.column) == 1 {
		i, err := strconv.Atoi(opt.column)
		if err == nil {
			ignoreCols = append(ignoreCols, i)
		}
	}

	if strings.Contains(opt.column, ",") {
		ignores := strings.Split(opt.column, ",")
		for _, ig := range ignores {
			i, err := strconv.Atoi(ig)
			if err != nil {
				ignoreCols = append(ignoreCols, i)
			}
		}
	}
	fmt.Println(opt.column)
	fmt.Println(ignoreCols)

	// var seen []string
	seen := make(map[string]bool)
	var cleanData []string

	for _, line := range data {
		lineData := strings.Split(line, opt.delimiter)
		if opt.verbose {
			fmt.Printf("\033[1;0m --> Raw: %v \033[1;0m\n", lineData)
		}
		// var compareData []string
		var compareData string

		// strip ignore column
		for _, i := range ignoreCols {
			if i == -1 {
				compareData = line
				continue
			}

			if i-1 >= len(lineData) {
				continue
			}

			if lineData[i] != "" {
				compareData += lineData[i]
			}
		}
		if opt.verbose {
			fmt.Printf("\033[1;32m ===> compareData: %v \033[1;37m\n", compareData)
		}
		// stripped := strings.Join(lineData, opt.delimiter)

		if _, ok := seen[compareData]; !ok {
			seen[compareData] = true
			fmt.Println(line)
			// fmt.Printf("--> %v \n", stripped)
			cleanData = append(cleanData, line)
		}

	}

}

func diff(data string, threshold float64) {

}
