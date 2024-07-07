package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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
		Timeout:   5 * time.Second, // Default timeout reduced to 5 seconds
	}
	return client, nil
}

// Function to check if URL returns HTTP 200 OK for both http and https
func checkURL(ctx context.Context, url string, client *http.Client, retries int, httpsOnly bool) (string, bool) {
	for i := 0; i <= retries; i++ {
		// Try with http
		if !httpsOnly {
			httpURL := "http://" + url
			req, err := http.NewRequestWithContext(ctx, "GET", httpURL, nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					return httpURL, true
				}
			}
		}

		// Try with https
		httpsURL := "https://" + url
		req, err := http.NewRequestWithContext(ctx, "GET", httpsURL, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return httpsURL, true
			}
		}

		time.Sleep(1 * time.Second) // Exponential backoff can be added here
	}
	return "", false
}

func processBlock(ctx context.Context, block []string, clients []*http.Client, retries int, httpsOnly bool, bar *progressbar.ProgressBar, writer *bufio.Writer, totalProcessed *uint64, liveCount *uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, url := range block {
		client := clients[rand.Intn(len(clients))]
		fullURL, alive := checkURL(ctx, url, client, retries, httpsOnly)
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

func main() {
	// Display script name, version and author
	fmt.Println("AliveHunter v0.9")
	color.New(color.FgHiYellow).Println("Made with love by Albert.C")

	// Parse command line arguments
	inputFile := flag.String("l", "", "File containing URLs to check")
	proxyFile := flag.String("p", "", "File containing proxy list (optional)")
	retries := flag.Int("r", 2, "Number of retries for failed requests") // Reduced retries to 2
	timeout := flag.Int("t", 5, "Timeout for HTTP requests in seconds")  // Reduced timeout to 5 seconds
	maxBlocks := flag.Int("b", 1000, "Maximum number of blocks to divide") // Maximum number of blocks
	httpsOnly := flag.Bool("https", false, "Check only HTTPS URLs")
	help := flag.Bool("h", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Usage: go run AliveHunter.go -l url.txt [-p proxy.txt] [-r retries] [-t timeout] [-b maxBlocks] [--https]")
		fmt.Println("\nOptions:")
		fmt.Println("  -l string")
		fmt.Println("        File containing URLs to check (required)")
		fmt.Println("  -p string")
		fmt.Println("        File containing proxy list (optional)")
		fmt.Println("  -r int")
		fmt.Println("        Number of retries for failed requests (default 2)")
		fmt.Println("  -t int")
		fmt.Println("        Timeout for HTTP requests in seconds (default 5)")
		fmt.Println("  -b int")
		fmt.Println("        Maximum number of blocks to divide (default 1000)")
		fmt.Println("  --https")
		fmt.Println("        Check only HTTPS URLs")
		fmt.Println("  -h    Show help message")
		fmt.Println("\nExamples:")
		fmt.Println("  go run AliveHunter.go -l url.txt")
		fmt.Println("  go run AliveHunter.go -l url.txt -p proxy.txt")
		fmt.Println("  go run AliveHunter.go -l url.txt -r 5")
		fmt.Println("  go run AliveHunter.go -l url.txt -t 15")
		fmt.Println("  go run AliveHunter.go -l url.txt -b 100")
		fmt.Println("  go run AliveHunter.go -l url.txt --https")
		fmt.Println("\nMake sure to install the necessary dependencies with:")
		fmt.Println("  go get github.com/fatih/color")
		fmt.Println("  go get github.com/schollz/progressbar/v3")
		fmt.Println("\nYou can also use proxychains for multi-node proxying:")
		fmt.Println("  proxychains go run AliveHunter.go -l url.txt")
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

	// Derive output file name
	outputFile := strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + "_alive.txt"

	// Display saving results message
	fmt.Printf("Saving the results to %s\n", outputFile)

	// Step 1: Read URLs from file
	urls, totalURLs, err := readURLsFromFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading URLs from file: %v\n", err)
		return
	}

	// Calculate number of blocks
	numBlocks := *maxBlocks
	if totalURLs < *maxBlocks {
		numBlocks = totalURLs
	}

	// Divide URLs into blocks
	blocks := divideIntoBlocks(urls, numBlocks)

	var totalProcessed uint64
	var liveCount uint64

	// Open the output file
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// Create default HTTP client
	defaultClient := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
	}
	clients := []*http.Client{defaultClient}

	// Create clients with proxies if available
	for _, proxy := range proxies {
		proxyClient, err := createClientWithProxy(proxy)
		if err == nil {
			clients = append(clients, proxyClient)
		}
	}

	// Create a progress bar
	bar := progressbar.NewOptions(totalURLs,
		progressbar.OptionSetDescription("Checking URLs..."),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false))

	// Signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Goroutine to process each block
	var wg sync.WaitGroup
	for _, block := range blocks {
		time.Sleep(1 * time.Second) // Small delay between blocks to avoid IP blocking
		wg.Add(1)
		go processBlock(ctx, block, clients, *retries, *httpsOnly, bar, writer, &totalProcessed, &liveCount, &wg)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Printf("\nTotal URLs processed: %d\n", totalProcessed)
	fmt.Printf("Live URLs found: %d\n", liveCount)

	fmt.Printf("Live URLs have been written to %s\n", outputFile)
}
