package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Function to read URLs from a file and return the total count and slice of URLs
func readURLsFromFile(filePath string) ([]string, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return urls, count, nil
}

// Function to divide URLs into blocks dynamically
func divideIntoBlocks(urls []string, numBlocks int) [][]string {
	var blocks [][]string
	blockSize := len(urls) / numBlocks
	for i := 0; i < len(urls); i += blockSize {
		end := i + blockSize
		if end > len(urls) {
			end = len(urls)
		}
		blocks = append(blocks, urls[i:end])
	}
	return blocks
}

// Function to read proxies from a file
func readProxiesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return proxies, nil
}

// Function to create an HTTP client with proxy
func createClientWithProxy(proxyURL string) (*http.Client, error) {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second, // Increased timeout for slow servers
	}
	return client, nil
}

// Function to check if URL returns HTTP 200 OK for both http and https
func checkURL(ctx context.Context, url string, client *http.Client, retries int, httpsOnly bool, logger *log.Logger) (string, bool) {
	for i := 0; i <= retries; i++ {
		if !httpsOnly {
			httpURL := "http://" + url
			req, err := http.NewRequestWithContext(ctx, "GET", httpURL, nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					return httpURL, true
				}
				if err != nil {
					logger.Printf("Error checking %s: %v\n", httpURL, err)
				}
			}
		}

		httpsURL := "https://" + url
		req, err := http.NewRequestWithContext(ctx, "GET", httpsURL, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return httpsURL, true
			}
			if err != nil {
				logger.Printf("Error checking %s: %v\n", httpsURL, err)
			}
		}

		time.Sleep(3 * time.Second) // Increased sleep time between retries
	}
	return "", false
}

func processBlock(ctx context.Context, block []string, clients []*http.Client, retries int, httpsOnly bool, bar *progressbar.ProgressBar, writer *bufio.Writer, totalProcessed *uint64, liveCount *uint64, wg *sync.WaitGroup, logger *log.Logger) {
	defer wg.Done()
	for _, url := range block {
		select {
		case <-ctx.Done():
			return
		default:
			client := clients[rand.Intn(len(clients))]
			fullURL, alive := checkURL(ctx, url, client, retries, httpsOnly, logger)
			if alive {
				writer.WriteString(fullURL + "\n")
				writer.Flush()
				atomic.AddUint64(liveCount, 1)
			}
			currentProcessed := atomic.AddUint64(totalProcessed, 1)
			bar.Describe(fmt.Sprintf("Checking URLs... (%d), Found alive: %d", currentProcessed, atomic.LoadUint64(liveCount)))
			bar.Add(1)
		}
	}
}

func main() {
	startTime := time.Now()
	fmt.Println("AliveHunter v1.0")
	color.New(color.FgHiYellow).Println("Made with love by Albert.C")
	fmt.Printf("Script started at: %s\n", startTime.Format("2006-01-02 15:04:05"))

	inputFile := flag.String("l", "", "File containing URLs to check (required)")
	outputFile := flag.String("o", "", "Output file to save the results (optional)")
	proxyFile := flag.String("p", "", "File containing proxy list (optional)")
	retries := flag.Int("r", 5, "Number of retries for failed requests") // Increased retries
	timeout := flag.Int("t", 15, "Timeout for HTTP requests in seconds") // Increased timeout
	maxBlocks := flag.Int("b", 1000, "Maximum number of blocks to divide")
	httpsOnly := flag.Bool("https", false, "Check only HTTPS URLs")
	help := flag.Bool("h", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Usage: go run AliveHunter.go -l subdomainlist.txt [-o output.txt] [-p proxylist.txt] [-r retries] [-t timeout] [-b maxBlocks] [--https]")
		fmt.Println("\nOptions:")
		fmt.Println("  -l string")
		fmt.Println("        File containing URLs to check (required)")
		fmt.Println("  -o string")
		fmt.Println("        Output file to save the results (optional, default is <input_file>_alive.txt)")
		fmt.Println("  -p string")
		fmt.Println("        File containing proxy list (optional)")
		fmt.Println("  -r int")
		fmt.Println("        Number of retries for failed requests (default 5)")
		fmt.Println("  -t int")
		fmt.Println("        Timeout for HTTP requests in seconds (default 15)")
		fmt.Println("  -b int")
		fmt.Println("        Maximum number of blocks to divide (default 1000)")
		fmt.Println("  --https")
		fmt.Println("        Check only HTTPS URLs")
		fmt.Println("  -h    Show help message")
		fmt.Println("\nExamples:")
		fmt.Println("  subfinder -d example.com --silent -o subdomainlist.txt && go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt")
		fmt.Println("  subfinder -d example.com --silent -o subdomainlist.txt && go run AliveHunter.go -l subdomainlist.txt")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt -p proxylist.txt")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt -r 5")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt -t 15")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt -b 100")
		fmt.Println("  go run AliveHunter.go -l subdomainlist.txt --https")
		fmt.Println("\nMake sure to install the necessary dependencies with:")
		fmt.Println("  go get github.com/fatih/color")
		fmt.Println("  go get github.com/schollz/progressbar/v3")
		fmt.Println("\nYou can also use proxychains for multi-node proxying:")
		fmt.Println("  proxychains go run AliveHunter.go -l subdomainlist.txt")
		return
	}

	if *inputFile == "" {
		fmt.Println("Please specify the input file using the -l flag. Use -h for help.")
		return
	}

	var proxies []string
	var err error
	if *proxyFile != "" {
		proxies, err = readProxiesFromFile(*proxyFile)
		if err != nil {
			fmt.Printf("Error reading proxies from file: %v\n", err)
			return
		}
	}

	outFileName := *outputFile
	if outFileName == "" {
		outFileName = strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + "_alive.txt"
	}

	fmt.Printf("Saving the results to %s\n", outFileName)

	urls, totalURLs, err := readURLsFromFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading URLs: %v\n", err)
		return
	}

	if totalURLs == 0 {
		fmt.Println("No URLs to process. Exiting.")
		return
	}

	numBlocks := *maxBlocks
	if totalURLs < *maxBlocks {
		numBlocks = totalURLs
	}

	if numBlocks == 0 {
		numBlocks = 1
	}

	blocks := divideIntoBlocks(urls, numBlocks)

	var totalProcessed uint64
	var liveCount uint64

	file, err := os.Create(outFileName)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	logFile, err := os.Create("error_log.txt")
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()
	logger := log.New(logFile, "ERROR: ", log.LstdFlags)

	defaultClient := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
	}
	clients := []*http.Client{defaultClient}

	for _, proxy := range proxies {
		proxyClient, err := createClientWithProxy(proxy)
		if err == nil {
			clients = append(clients, proxyClient)
		}
	}

	bar := progressbar.NewOptions(totalURLs,
		progressbar.OptionSetDescription("Checking URLs..."),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for _, block := range blocks {
		time.Sleep(1 * time.Second)
		wg.Add(1)
		go processBlock(ctx, block, clients, *retries, *httpsOnly, bar, writer, &totalProcessed, &liveCount, &wg, logger)
	}

	wg.Wait()

	endTime := time.Now()
	fmt.Printf("\nTotal URLs processed: %d\n", totalProcessed)
	fmt.Printf("Live URLs found: %d\n", liveCount)
	fmt.Printf("Live URLs have been written to %s\n", outFileName)
	fmt.Printf("Script finished at: %s\n", endTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total execution time: %s\n", endTime.Sub(startTime))
}
