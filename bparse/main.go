package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/csv"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type blob struct {
	Text   string `xml:",chardata"`
	Base64 bool   `xml:"base64,attr"`
}

// raw returns the bytes Burp stored, base64-decoded when flagged.
func (b blob) raw() []byte {
	if !b.Base64 {
		return []byte(b.Text)
	}
	d, err := base64.StdEncoding.DecodeString(b.Text)
	if err != nil {
		return []byte(b.Text)
	}
	return d
}

type Item struct {
	URL  string `xml:"url"`
	Host struct {
		Text string `xml:",chardata"`
		IP   string `xml:"ip,attr"`
	} `xml:"host"`
	Port           string `xml:"port"`
	Protocol       string `xml:"protocol"`
	Method         string `xml:"method"`
	Path           string `xml:"path"`
	Request        blob   `xml:"request"`
	Status         string `xml:"status"`
	Responselength string `xml:"responselength"`
	Mimetype       string `xml:"mimetype"`
	Response       blob   `xml:"response"`
}

type Items struct {
	Item []Item `xml:"item"`
}

var (
	burpFile string
	output   string
	noBody   bool
	flat     bool
	decode   bool
	dump     bool
	top      int
)

const usage = `Usage:
  bparse [flags] burp.xml        export to CSV (or -f: one base64 request per line)
  bparse stat [flags] burp.xml   print statistics

Flags:
`

func main() {
	args := os.Args[1:]
	stat := len(args) > 0 && args[0] == "stat"
	if stat {
		args = args[1:]
	}
	flag.StringVar(&burpFile, "i", "", "Burp file (default is first positional argument)")
	flag.StringVar(&output, "o", "", "Output file (default out.csv, or out.txt with -f)")
	flag.BoolVar(&noBody, "n", false, "Don't include request/response bodies")
	flag.BoolVar(&decode, "d", false, "Write bodies as decoded text instead of base64 (gzip/chunked handled)")
	flag.BoolVar(&dump, "r", false, "Print every request/response fully decoded to stdout (for grep)")
	flag.BoolVar(&flat, "f", false, "Write raw requests, base64, one per line")
	flag.IntVar(&top, "top", 10, "stat: number of top hosts/URLs to show")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage); flag.PrintDefaults() }
	// flag stops at the first positional arg; keep parsing so flags may follow the file name
	var pos []string
	for {
		flag.CommandLine.Parse(args)
		if flag.NArg() == 0 {
			break
		}
		pos = append(pos, flag.Arg(0))
		args = flag.Args()[1:]
	}
	if burpFile == "" && len(pos) > 0 {
		burpFile = pos[0]
	}
	if burpFile == "" {
		flag.Usage()
		os.Exit(2)
	}

	if output == "" {
		output = "out.csv"
		if flat {
			output = "out.txt"
		}
	}
	src := stream(burpFile)
	switch {
	case stat:
		fatalIf(printStats(src))
	case dump:
		fatalIf(writeDump(src))
	case flat:
		fatalIf(writeFlat(src))
	default:
		fatalIf(writeCSV(src))
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }
func fatalIf(err error) {
	if err != nil {
		fatal(err)
	}
}

// source calls fn for every <item>, decoding one at a time so memory stays flat on huge exports.
type source func(fn func(Item)) error

func stream(name string) source {
	if strings.HasPrefix(name, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			name = filepath.Join(h, name[2:])
		}
	}
	return func(fn func(Item)) error {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		d := xml.NewDecoder(bufio.NewReader(f))
		for {
			tok, err := d.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("parsing Burp XML: %w", err)
			}
			if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "item" {
				var it Item
				if err := d.DecodeElement(&it, &se); err != nil {
					return fmt.Errorf("parsing Burp XML: %w", err)
				}
				fn(it)
			}
		}
	}
}

func writeFlat(src source) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	n := 0
	err = src(func(it Item) {
		fmt.Fprintln(w, base64.StdEncoding.EncodeToString(it.Request.raw()))
		n++
	})
	if err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Printf("wrote %d requests to %s\n", n, output)
	return nil
}

// writeDump prints each item as decoded plain text: headers as-is, body un-chunked/un-gzipped.
func writeDump(src source) error {
	w := bufio.NewWriter(os.Stdout)
	n := 0
	err := src(func(it Item) {
		n++
		fmt.Fprintf(w, "=== #%d %s %s [%s] ===\n--- request ---\n", n, it.Method, it.URL, it.Status)
		w.Write(message(it.Request.raw(), false))
		fmt.Fprint(w, "\n--- response ---\n")
		w.Write(message(it.Response.raw(), true))
		fmt.Fprint(w, "\n\n")
	})
	if err != nil {
		return err
	}
	return w.Flush()
}

func message(raw []byte, isResp bool) []byte {
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if head, _, ok := bytes.Cut(raw, []byte(sep)); ok {
			return append(append(head, sep...), body(raw, isResp)...)
		}
	}
	return raw
}

func writeCSV(src source) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f) // handles commas, quotes and newlines itself
	hdr := []string{"URL", "Method", "Host", "Path", "Protocol", "IP:Port", "Status", "Responselength", "Mimetype"}
	if !noBody {
		hdr = append(hdr, "RequestBody", "ResponseBody")
	}
	w.Write(hdr)
	n := 0
	err = src(func(it Item) {
		n++
		row := []string{it.URL, it.Method, it.Host.Text, it.Path, it.Protocol,
			it.Host.IP + ":" + it.Port, it.Status, it.Responselength, it.Mimetype}
		if !noBody {
			row = append(row, encodeBody(it.Request.raw(), false), encodeBody(it.Response.raw(), true))
		}
		w.Write(row)
	})
	if err != nil {
		return err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	fmt.Printf("wrote %d rows to %s\n", n, output)
	return nil
}

func encodeBody(raw []byte, isResp bool) string {
	b := body(raw, isResp)
	if decode {
		return string(b)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// body extracts the HTTP message body, undoing chunked and gzip encoding.
// Malformed messages (common in pentest traffic) fall back to a header/body split.
func body(raw []byte, isResp bool) []byte {
	if len(raw) == 0 {
		return nil
	}
	r := bufio.NewReader(bytes.NewReader(raw))
	var b io.Reader
	var h http.Header
	if isResp {
		resp, err := http.ReadResponse(r, nil)
		if err != nil {
			return splitBody(raw)
		}
		b, h = resp.Body, resp.Header
	} else {
		req, err := http.ReadRequest(r)
		if err != nil {
			return splitBody(raw)
		}
		b, h = req.Body, req.Header
	}
	if h.Get("Content-Encoding") == "gzip" {
		if gz, err := gzip.NewReader(b); err == nil {
			b = gz
		}
	}
	out, _ := io.ReadAll(b) // keep partial data on truncated messages
	return out
}

func splitBody(raw []byte) []byte {
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if _, after, ok := bytes.Cut(raw, []byte(sep)); ok {
			return after
		}
	}
	return nil
}

func printStats(src source) error {
	method, status, mime, proto, host, url := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	var total, bytesTotal, withResp int
	err := src(func(it Item) {
		total++
		method[it.Method]++
		status[it.Status]++
		mime[it.Mimetype]++
		proto[it.Protocol]++
		host[it.Host.Text]++
		url[it.URL]++
		n, _ := strconv.Atoi(it.Responselength)
		bytesTotal += n
		if it.Response.Text != "" {
			withResp++
		}
	})
	if err != nil {
		return err
	}
	fmt.Printf("Requests:       %d\n", total)
	fmt.Printf("With response:  %d\n", withResp)
	fmt.Printf("Unique hosts:   %d\n", len(host))
	fmt.Printf("Unique URLs:    %d\n", len(url))
	fmt.Printf("Response bytes: %d\n", bytesTotal)
	section("Methods", method, 0)
	section("Status codes", status, 0)
	section("Protocols", proto, 0)
	section("MIME types", mime, 0)
	section("Top hosts", host, top)
	section("Top URLs", url, top)
	return nil
}

func section(title string, m map[string]int, limit int) {
	type kv struct {
		k string
		n int
	}
	s := make([]kv, 0, len(m))
	for k, n := range m {
		if k == "" {
			k = "(none)"
		}
		s = append(s, kv{k, n})
	}
	sort.Slice(s, func(i, j int) bool {
		if s[i].n != s[j].n {
			return s[i].n > s[j].n
		}
		return s[i].k < s[j].k
	})
	if limit > 0 && len(s) > limit {
		s = s[:limit]
	}
	fmt.Printf("\n%s:\n", title)
	for _, e := range s {
		fmt.Printf("  %6d  %s\n", e.n, e.k)
	}
}
