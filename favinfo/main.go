package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/twmb/murmur3"
)

var (
	jsonOutput  bool
	concurrency int
)

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 20, "Set the concurrency level")
	flag.BoolVar(&jsonOutput, "json", false, "Show Output as Json format")
	flag.Parse()

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				GetFavHash(job)
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	go func() {
		for sc.Scan() {
			raw := strings.TrimSpace(sc.Text())
			if err := sc.Err(); err == nil && raw != "" {
				jobs <- raw
			}
		}
		close(jobs)
	}()
	wg.Wait()
}

func GetFavHash(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return ""
	}
	hashURL := fmt.Sprintf("%v://%v/favicon.ico", u.Scheme, u.Host)
	err, data := BigResponseReq(hashURL)
	if err != nil {
		hashURL = URL
		err, data = BigResponseReq(hashURL)
		if data == "" {
			return ""
		}
	}

	hashedFav := Mmh3Hash32(StandBase64([]byte(data)))
	fmt.Printf("%s,%s\n", hashURL, hashedFav)
	return hashedFav
}

func Mmh3Hash32(raw []byte) string {
	h32 := murmur3.New32()
	_, err := h32.Write(raw)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", int32(h32.Sum32()))
}

// StandBase64 base64 from bytes
func StandBase64(data []byte) []byte {
	raw := base64.StdEncoding.EncodeToString(data)
	var buffer bytes.Buffer
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

func BigResponseReq(baseUrl string) (error, string) {
	fmt.Fprintf(os.Stderr, "sending get %s\n", baseUrl)

	client := &http.Client{
		Timeout: time.Duration(10*3) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Second * 60,
			}).DialContext,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true, Renegotiation: tls.RenegotiateOnceAsClient},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	req, _ := http.NewRequest("GET", baseUrl, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sending get %v\n", err)
		return err, ""
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sending get %v\n", err)
		return err, ""
	}

	if resp.StatusCode != 200 {
		fmt.Println(string(content))
		return fmt.Errorf("no favicon"), string(content)
	}

	return nil, string(content)
}
