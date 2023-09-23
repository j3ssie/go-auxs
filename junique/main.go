package main

import (
    "bufio"
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/projectdiscovery/hmap/store/hybrid"
    "github.com/tidwall/gjson"
)

// cat raw.json | junique -k 'hash' | sort -u > unique-hosts.json

func main() {
    var jKey string
    flag.StringVar(&jKey, "k", "", "Json key for unique ( https://github.com/tidwall/gjson )")
    flag.Parse()

    hm, err := hybrid.New(hybrid.DefaultDiskOptions)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to init map")
        os.Exit(1)
    }
    defer hm.Close()
    sc := bufio.NewScanner(os.Stdin)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" {
            continue
        }
        jValue := gjson.Get(line, jKey).String()
        if jValue == "" {
            continue
        }
        if _, exist := hm.Get(jValue); !exist {
            hm.Set(jValue, []byte("0"))
            fmt.Println(line)
        }
    }

}
