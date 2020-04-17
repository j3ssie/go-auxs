package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Do screenshot from list of URLs
// cat /tmp/list_of_urls.txt | qscreenshot -o ouput

const (
	QUALITY = 90
)

var (
	output      string
	indexFile   string
	concurrency int
	verbose     bool
	timeout     int
	imgWidth    int
	imgHeight   int
)

func main() {
	// cli arguments
	flag.IntVar(&concurrency, "c", 10, "Set the concurrency level")
	flag.StringVar(&output, "o", "screen", "Output Directory")
	flag.StringVar(&indexFile, "s", "", "Summary File")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.IntVar(&timeout, "timeout", 10, "screenshot timeout")
	flag.IntVar(&imgHeight, "height", 0, "height screenshot")
	flag.IntVar(&imgWidth, "width", 0, "width screenshot")
	flag.Parse()

	// prepare output
	if indexFile == "" {
		indexFile = path.Join(output, "screen-summary.txt")
	}
	err := os.MkdirAll(output, 0750)
	if err != nil {
		log.Fatal("Can't create output directory")
	}

	var wg sync.WaitGroup
	jobs := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				log.Printf("[*] processing: %v", job)
				imgScreen := doScreenshot(job)
				if imgScreen != "" {
					log.Printf("[*] Store image: %v %v", job, imgScreen)
					sum := fmt.Sprintf("%v - %v", job, imgScreen)
					appendSummary(indexFile, sum)
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

	info, err := os.Stat(indexFile)
	if !os.IsNotExist(err) && info.IsDir() {
		log.Printf("[+] Store summary in: %v", indexFile)
	}
}

func doScreenshot(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	imageDir := path.Join(output, u.Hostname())
	os.MkdirAll(imageDir, 0750)
	imageScreen := path.Join(imageDir, fmt.Sprintf("%v.png", strings.Replace(raw, "/", "_", -1)))

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	// create context
	allocCtx, bcancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer bcancel()
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// capture screenshot of an element
	var buf []byte
	err = chromedp.Run(ctx, fullScreenshot(raw, QUALITY, &buf))
	// clean chromedp-runner folder
	cleanUp()
	if err != nil {
		log.Printf("[-] screen err: %v", raw)
		return ""
	}

	// write image
	if err := ioutil.WriteFile(imageScreen, buf, 0644); err != nil {
		log.Printf("[-] screen err: %v", raw)
	}
	return imageScreen

}

// fullScreenshot takes a screenshot of the entire browser viewport.
// Liberally copied from puppeteer's source.
// Note: this will override the viewport emulation settings.
func fullScreenshot(urlstr string, quality int64, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// get layout metrics
			_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}

			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))
			//imgWidth       int
			//imgHeight      int
			if imgWidth != 0 && imgHeight != 0 {
				width = int64(imgWidth)
				height = int64(imgHeight)
			}

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}

			// capture screenshot
			*res, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  float64(width),
					Height: float64(height),
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
}

func cleanUp() {
	tmpFolder := path.Join(os.TempDir(), "chromedp-runner*")
	if _, err := os.Stat("/tmp/"); !os.IsNotExist(err) {
		tmpFolder = path.Join("/tmp/", "chromedp-runner*")
	}
	junks, err := filepath.Glob(tmpFolder)
	if err != nil {
		return
	}
	for _, junk := range junks {
		os.RemoveAll(junk)
	}
}

// appendSummary append string to a file
func appendSummary(filename string, data string) (string, error) {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	if _, err := f.Write([]byte(data + "\n")); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return filename, nil
}
