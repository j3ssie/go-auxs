package main

import "testing"

func Test_getCurrentCC(t *testing.T) {
    result, err := getCurrentCC()
    if err != nil {
        t.Failed()
    }
    t.Log(result)
}

func Test_downloadWaybackResults(t *testing.T) {
    RawOutput = true
    err := downloadWaybackResults("http://web.archive.org/cdx/search/cdx?url=yahoo.com/*&output=json&collapse=urlkey")
    if err != nil {
        t.Fatal(err)
    }
}

func Test_getCommonCrawlURLs(t *testing.T) {
    RawOutput = true
    err := getCommonCrawlURLs("yahoo.com")
    if err != nil {
        t.Fatal(err)
    }
}
