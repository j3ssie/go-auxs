package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type Items struct {
	XMLName     xml.Name `xml:"items"`
	Text        string   `xml:",chardata"`
	BurpVersion string   `xml:"burpVersion,attr"`
	ExportTime  string   `xml:"exportTime,attr"`
	Item        []struct {
		Text string `xml:",chardata"`
		Time string `xml:"time"`
		URL  string `xml:"url"`
		Host struct {
			Text string `xml:",chardata"`
			Ip   string `xml:"ip,attr"`
		} `xml:"host"`
		Port      string `xml:"port"`
		Protocol  string `xml:"protocol"`
		Method    string `xml:"method"`
		Path      string `xml:"path"`
		Extension string `xml:"extension"`
		Request   struct {
			Text   string `xml:",chardata"`
			Base64 string `xml:"base64,attr"`
		} `xml:"request"`
		Status         string `xml:"status"`
		Responselength string `xml:"responselength"`
		Mimetype       string `xml:"mimetype"`
		Response       struct {
			Text   string `xml:",chardata"`
			Base64 string `xml:"base64,attr"`
		} `xml:"response"`
		Comment string `xml:"comment"`
	} `xml:"item"`
}

var (
	burpFile   string
	output     string
	isBase64   bool
	noBody     bool
	flat       bool
	stripComma bool
)

// Usage:
// bparse -o output.csv burp-file
// bparse -n -o output.csv burp-file
// bparse  -i burp-file -f -o output

func main() {
	// cli arguments
	flag.StringVar(&burpFile, "i", "", "Burp file (default is last argument)")
	flag.StringVar(&output, "o", "out", "Output file")
	flag.BoolVar(&noBody, "n", false, "Don't store body in csv output")
	flag.BoolVar(&isBase64, "b", true, "is Burp XML base64 encoded")
	flag.BoolVar(&flat, "f", false, "Store raw request base64 line by line")
	flag.BoolVar(&stripComma, "s", true, "Encode ',' in case it appear in data")
	flag.Parse()
	args := os.Args[1:]
	sort.Strings(args)
	if burpFile == "" {
		burpFile = args[len(args)-1]
	}
	content := GetFileContent(burpFile)
	if content == "" {
		fmt.Printf("failed to read content of %v \n", burpFile)
		return
	}

	// parsing content
	r := &Items{}
	err := xml.Unmarshal([]byte(content), r)
	if err != nil {
		fmt.Printf("failed to parse Burp XML file: %v \n", err)
		return
	}

	// URL, Host, Method, Path, IP:Port, Status, Responselength, Body
	csvHeader := []string{"URL", "Method", "Host", "Path", "Protocol", "IP:Port", "Status", "Responselength", "Body"}
	if noBody {
		csvHeader = []string{"URL", "Method", "Host", "Path", "Protocol", "IP:Port", "Status", "Responselength"}
	}
	csvData := [][]string{
		csvHeader,
	}
	if !flat {
		fmt.Println(strings.Join(csvHeader, ","))
	}

	var flatOutput []string
	// loop through data
	for _, item := range r.Item {
		if flat {
			flatOutput = append(flatOutput, item.Request.Text)
			fmt.Println(item.Request.Text)
			continue
		}
		if stripComma {
			if strings.Contains(item.URL, ",") {
				item.URL = strings.Replace(item.URL, ",", "%2c", -1)
			}
			if strings.Contains(item.Path, ",") {
				item.Path = strings.Replace(item.Path, ",", "%2c", -1)
			}
		}

		dest := fmt.Sprintf("%v:%v", item.Host.Ip, item.Port)
		data := []string{
			item.URL, item.Method, item.Host.Text, item.Path, item.Protocol, dest, item.Status, item.Responselength,
		}
		if !noBody {
			body := GetReqBody(item.Request.Text)
			data = []string{
				item.URL, item.Method, item.Host.Text, item.Path, item.Protocol, dest, item.Status, item.Responselength, body,
			}
		}
		csvData = append(csvData, data)
		fmt.Println(strings.Join(data, ","))
	}
	if flat {
		WriteToFile(output, strings.Join(flatOutput, "\n"))
		return
	}

	// write to CSV
	csvFile, err := os.Create(output)
	if err != nil {
		fmt.Printf("failed to write csv data: %s \n", err)
		return
	}
	csvWriter := csv.NewWriter(csvFile)
	for _, empRow := range csvData {
		_ = csvWriter.Write(empRow)
	}
	csvWriter.Flush()
	csvFile.Close()
}

// GetFileContent Reading file and return content of it
func GetFileContent(filename string) string {
	var result string
	if strings.Contains(filename, "~") {
		filename, _ = homedir.Expand(filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return result
	}
	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return result
	}
	return string(b)
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

// GetReqBody parse burp style request
func GetReqBody(raw string) string {
	var body string
	if isBase64 {
		raw, _ = Base64Decode(raw)
	}
	reader := bufio.NewReader(strings.NewReader(raw))
	parsedReq, err := http.ReadRequest(reader)
	if err != nil {
		return raw
	}
	rBody, _ := ioutil.ReadAll(parsedReq.Body)
	body = string(rBody)
	if strings.Contains(body, ",") || strings.Contains(body, "\n") {
		body = URLEncode(body)
	}
	body = Base64Encode(body)
	return body
}

// Base64Encode just Base64 Encode
func Base64Encode(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// URLEncode just URL Encode
func URLEncode(raw string) string {
	return url.QueryEscape(raw)
}

// Base64Decode just Base64 Encode
func Base64Decode(raw string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw, err
	}
	return string(data), nil
}
