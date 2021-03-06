package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"net/url"
	"os"
	"strings"
)

// Literally copied from: https://github.com/tomnomnom/unfurl
// with some improvements
var limit int

func main() {
	var unique bool
	flag.BoolVar(&unique, "u", false, "")
	flag.BoolVar(&unique, "unique", false, "")

	var verbose bool
	flag.BoolVar(&verbose, "v", false, "")
	flag.BoolVar(&verbose, "verbose", false, "")
	flag.IntVar(&limit, "l", 100, "limit size")

	flag.Parse()

	mode := flag.Arg(0)
	fmtStr := flag.Arg(1)

	procFn, ok := map[string]urlProc{
		"keys":    keys,
		"values":  values,
		"domains": domains,
		"paths":   paths,
		"format":  format,
	}[mode]

	if !ok {
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", mode)
		return
	}

	sc := bufio.NewScanner(os.Stdin)

	seen := make(map[string]bool)

	for sc.Scan() {
		u, err := url.Parse(sc.Text())
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "parse failure: %s\n", err)
			}
			continue
		}

		// some urlProc functions return multiple things,
		// so it's just easier to always get a slice and
		// loop over it instead of having two kinds of
		// urlProc functions.
		for _, val := range procFn(u, fmtStr) {

			// you do see empty values sometimes
			if val == "" {
				continue
			}

			if seen[val] && unique {
				continue
			}

			fmt.Println(val)

			// no point using up memory if we're outputting dupes
			if unique {
				seen[val] = true
			}
		}
	}

	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %s\n", err)
	}
}

type urlProc func(*url.URL, string) []string

func keys(u *url.URL, _ string) []string {
	out := make([]string, 0)
	for key, _ := range u.Query() {
		out = append(out, key)
	}
	return out
}

func values(u *url.URL, _ string) []string {
	out := make([]string, 0)
	for _, vals := range u.Query() {
		for _, val := range vals {
			out = append(out, val)
		}
	}
	return out
}

func domains(u *url.URL, f string) []string {
	return format(u, "%d")
}

func paths(u *url.URL, f string) []string {
	return format(u, "%p")
}

func format(u *url.URL, f string) []string {

	out := &bytes.Buffer{}
	inFormat := false
	for _, r := range f {

		if r == '%' && !inFormat {
			inFormat = true
			continue
		}

		if !inFormat {
			out.WriteRune(r)
			continue
		}

		switch r {
		case '%':
			out.WriteRune('%')

		case 's':
			out.WriteString(u.Scheme)
		case 'd':
			out.WriteString(u.Hostname())
		case 'P':
			out.WriteString(u.Port())
		case 'p':
			out.WriteString(u.EscapedPath())
		case 'q':
			out.WriteString(u.RawQuery)
		case 'f':
			out.WriteString(u.Fragment)
		case 'n':
			out.WriteRune('\n')
		case 'o':
			domain := u.Host
			org := u.Host
			suffix, ok := publicsuffix.PublicSuffix(domain)
			if ok {
				org = strings.Replace(domain, fmt.Sprintf(".%s", suffix), "", -1)
			} else {
				if strings.Contains(domain, ".") {
					parts := strings.Split(domain, ".")
					if len(parts) == 2 {
						org = parts[0]
					} else {
						org = parts[len(parts)-2]
					}
				}
			}
			out.WriteString(org)
		case 'D':
			var dots string
			if strings.Contains(u.Host, ".") {
				dots = strings.Join(strings.Split(u.Host, "."), "\n")
				dots = strings.Trim(dots, "\n")
			}
			out.WriteString(dots)
		// get paths but in lists
		case 'E':
			rPaths := u.EscapedPath()
			var paths string
			if strings.Contains(rPaths, "/") {
				for _, p := range strings.Split(rPaths, "/") {
					if len(p) < limit {
						paths += p + "\n"
					}
				}
				paths = strings.Trim(paths, "\n")
			}
			out.WriteString(paths)
		// get query but in lists
		case 'Q':
			rQueries := u.Query()
			var queries string
			if len(rQueries) > 0 {
				for k := range rQueries {
					queries += k + "\n"
				}
			}
			queries = strings.Trim(queries, "\n")
			out.WriteString(queries)
		default:
			// output untouched
			out.WriteRune('%')
			out.WriteRune(r)
		}

		inFormat = false
	}

	return []string{out.String()}
}

func init() {
	flag.Usage = func() {
		h := "Format URLs provided on stdin\n\n"

		h += "Usage:\n"
		h += "  purl [OPTIONS] [MODE] [FORMATSTRING]\n\n"

		h += "Options:\n"
		h += "  -u, --unique   Only output unique values\n"
		h += "  -v, --verbose  Verbose mode (output URL parse errors)\n\n"

		h += "Modes:\n"
		h += "  keys     Keys from the query string (one per line)\n"
		h += "  values   Values from the query string (one per line)\n"
		h += "  domains  The hostname (e.g. sub.example.com)\n"
		h += "  paths    The request path (e.g. /users)\n"
		h += "  format   Specify a custom format (see below)\n\n"

		h += "Format Directives:\n"
		h += "  %%  A literal percent character\n"
		h += "  %n  A new line character\n"
		h += "  %s  The request scheme (e.g. https)\n"
		h += "  %d  The domain (e.g. sub.example.com)\n"
		h += "  %o  The root domain (e.g. example)\n"
		h += "  %P  The port (e.g. 8080)\n"
		h += "  %p  The path (e.g. /users)\n"
		h += "  %e  The sinlge paths (e.g. user)\n"
		h += "  %q  The raw query string (e.g. a=1&b=2)\n"
		h += "  %f  The page fragment (e.g. page-section)\n\n"

		h += "Examples:\n"
		h += "  cat urls.txt | purl keys\n"
		h += "  cat urls.txt | purl format %s://%d%p?%q\n"
		h += "  cat urls.txt | purl format %D%n%E%n%Q\n"

		fmt.Fprint(os.Stderr, h)
	}
}
