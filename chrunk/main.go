package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Extend the IP range by CIDR
// Usage: chrunk -i /tmp/really_big_file.txt -cmd 'echo "--> {}"'

var logger = logrus.New()

var (
	clean       bool
	concurrency int
	part        int
	size        int
	filename    string
	command     string
	output      string
	prefix      string
	outDir      string
)

func main() {
	logger = &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.InfoLevel,
		Formatter: &prefixed.TextFormatter{
			ForceColors:     true,
			ForceFormatting: true,
		},
	}

	// cli arguments
	flag.BoolVar(&clean, "clean", false, "Clean junk file after done")
	flag.IntVar(&part, "p", 0, "Number of parts to split")
	flag.IntVar(&size, "s", 10000, "Number of lines to split file")
	flag.StringVar(&filename, "i", "", "Input file to split")
	flag.StringVar(&prefix, "prefix", "", "Prefix output filename")
	flag.StringVar(&outDir, "o", "", "Output foldeer contains list of filename")
	flag.IntVar(&concurrency, "c", 1, "Set the concurrency level")
	flag.StringVar(&command, "cmd", "", "Command to run after chunked content")
	flag.Parse()

	if outDir == "" {
		outDir = path.Join(os.TempDir(), "chrunk-data")
		os.MkdirAll(outDir, 0755)
		logger.Infof("Set output folder: %v", outDir)
	}

	// input as stdin
	if filename == "" {
		var rawInput []string
		stat, _ := os.Stdin.Stat()
		// detect if anything came from std
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				url := strings.TrimSpace(sc.Text())
				if err := sc.Err(); err == nil && url != "" {
					rawInput = append(rawInput, url)
				}
			}
		}

		filename = path.Join(outDir, fmt.Sprintf("raw-%v", RandomString(8)))
		logger.Infof("Write stdin data to: %v", filename)
		WriteToFile(filename, strings.Join(rawInput, "\n"))
	}

	if filename == "" {
		logger.Panic("No input provided")
		os.Exit(-1)
	}

	if prefix == "" {
		prefix = strings.TrimSuffix(path.Base(filename), path.Ext(filename))
		logger.Infof("Set prefix output: %v", prefix)
	}

	var divided [][]string
	// really split file here
	if part == 0 {
		divided = ChunkFileBySize(filename, size)
	} else {
		divided = ChunkFileByPart(filename, part)
	}

	var chunkFiles []string
	// write data
	logger.Infof("Split input to %v parts", len(divided))
	for index, chunk := range divided {
		outName := path.Join(outDir, fmt.Sprintf("%v-%v", prefix, index))
		if command == "" {
			fmt.Println(outName)
		}
		WriteToFile(outName, strings.Join(chunk, "\n"))
		chunkFiles = append(chunkFiles, outName)
	}

	var commands []string
	if command != "" {
		// run command here
		for index, chunkFile := range chunkFiles {
			cmd := command
			cmd = strings.Replace(cmd, "{}", chunkFile, -1)
			cmd = strings.Replace(cmd, "{#}", fmt.Sprintf("%d", index), -1)
			// Execution(cmd)
			commands = append(commands, cmd)
		}
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				Execution(job)
			}
		}()
	}

	for _, command := range commands {
		jobs <- command
	}
	close(jobs)
	wg.Wait()

	// cleanup tmp data
	if clean || command != "" {
		logger.Infof("Clean up tmp data")
		for _, chunkFile := range chunkFiles {
			os.RemoveAll(chunkFile)
		}
	}
}

// ChunkFileByPart chunk file to multiple part
func ChunkFileByPart(source string, chunk int) [][]string {
	var divided [][]string
	data := ReadingLines(source)
	if len(data) <= 0 || chunk > len(data) {
		if len(data) > 0 {
			divided = append(divided, data)
		}
		return divided
	}

	chunkSize := (len(data) + chunk - 1) / chunk
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		divided = append(divided, data[i:end])
	}
	return divided
}

// ChunkFileBySize chunk file to multiple part
func ChunkFileBySize(source string, chunk int) [][]string {
	var divided [][]string
	data := ReadingLines(source)
	if len(data) <= 0 || chunk > len(data) {
		if len(data) > 0 {
			divided = append(divided, data)
		}
		return divided
	}

	chunkSize := chunk
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		divided = append(divided, data[i:end])
	}
	return divided
}

// ReadingLines Reading file and return content as []string
func ReadingLines(filename string) []string {
	var result []string
	if strings.HasPrefix(filename, "~") {
		filename, _ = homedir.Expand(filename)
	}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return result
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := scanner.Text()
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

// WriteToFile write string to a file
func WriteToFile(filename string, data string) (string, error) {
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.WriteString(file, data+"\n")
	if err != nil {
		return "", err
	}
	return filename, file.Sync()
}

// Execution Run a command
func Execution(cmd string) (string, error) {
	command := []string{
		"bash",
		"-c",
		cmd,
	}
	var output string
	logger.Infof("Execute: %s", cmd)
	realCmd := exec.Command(command[0], command[1:]...)
	// output command output to std too
	cmdReader, _ := realCmd.StdoutPipe()
	scanner := bufio.NewScanner(cmdReader)
	var out string
	go func() {
		for scanner.Scan() {
			out += scanner.Text()
			fmt.Println(scanner.Text())
		}
	}()
	if err := realCmd.Start(); err != nil {
		return "", err
	}
	if err := realCmd.Wait(); err != nil {
		return "", err
	}
	return output, nil
}

// RandomString return a random string with length
func RandomString(n int) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var letter = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[seededRand.Intn(len(letter))]
	}
	return string(b)
}
