package main

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScrapeRanges(t *testing.T) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			TLSClientConfig: &tls.Config{
				Renegotiation:      tls.RenegotiateOnceAsClient,
				InsecureSkipVerify: true,
			},
		},

		Timeout: time.Duration(30) * time.Second,
	}

	t.Run("cloudflare", func(t *testing.T) {
		out, err := scrapeCloudflare(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape cloudflare")
	})
	t.Run("incapsula", func(t *testing.T) {
		out, err := scrapeIncapsula(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape incapsula")
	})
	t.Run("akamai", func(t *testing.T) {
		out, err := scrapeAkamai(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape akamai")
	})
	t.Run("sucuri", func(t *testing.T) {
		out, err := scrapeSucuri(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape sucuri")
	})
	t.Run("projectdiscovery", func(t *testing.T) {
		out, err := scrapeProjectDiscovery(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape projectdiscovery")
	})
	t.Run("cloudfront", func(t *testing.T) {
		out, err := scrapeCloudfront(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape cloudfront")
	})
	t.Run("fastly", func(t *testing.T) {
		out, err := scrapeFastly(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape fastly")
	})
	t.Run("maxcdn", func(t *testing.T) {
		out, err := scrapeMaxCDN(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape maxcdn")
	})
	t.Run("ddosguard", func(t *testing.T) {
		out, err := scrapeDDOSGuard(httpClient)
		t.Log(out)
		require.Nil(t, err, "Could not scrape ddosguard")
	})
}
