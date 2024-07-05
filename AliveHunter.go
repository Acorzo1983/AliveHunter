package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Function to read URLs from a file
func readURLsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
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
		Timeout:   10 * time.Second,
	}
	return client, nil
}

// Function to check if URL returns HTTP 200 OK for both http and https
func checkURL(url string, client *http.Client) bool {
	// Try with http
	httpURL := "http://" + url
	resp, err := client.Get(httpURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		return true
	}

	// Try with https
	httpsURL := "https://" + url
	resp, err = client.Get(httpsURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		return true
	}

	return false
}

func main() {
	// Display script name, version and author
	fmt.Println("AliveHunter v0.9")
	color.New(color.FgHiYellow).Println("Made with love by Albert.C")

	// Parse command line arguments
	inputFile := flag.String("l", "", "File containing URLs to check")
	proxyFile := flag.String("p", "", "File containing proxy list (optional)")
	help := flag.Bool("h", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Usage: go run AliveHunter.go -l url.txt [-p proxy.txt]")
		fmt.Println("Options:")
		fmt.Println("  -l string")
		fmt.Println("        File containing URLs to check")
		fmt.Println("  -p string")
		fmt.Println("        File containing proxy list (optional)")
		fmt.Println("  -h    Show help message")
		fmt.Println("Make sure to install the necessary dependencies with:")
		fmt.Println("  go get github.com/fatih/color")
		fmt.Println("  go get github.com/schollz/progressbar/v3")
		fmt.Println("You can also use proxychains for multi-node proxying:")
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
	urls, err := readURLsFromFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading URLs from file: %v\n", err)
		return
	}

	totalURLs := len(urls)
	var wg sync.WaitGroup
	var client *http.Client

	// Create default HTTP client
	defaultClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	clients := []*http.Client{defaultClient}

	// Create clients with proxies if available
	for _, proxy := range proxies {
		proxyClient, err := createClientWithProxy(proxy)
		if err == nil {
			clients = append(clients, proxyClient)
		}
	}

	urlsChan := make(chan string, totalURLs)

	// Open the output file
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	var totalProcessed uint64
	var liveCount uint64

	// Create a progress bar
	bar := progressbar.NewOptions(totalURLs,
		progressbar.OptionSetDescription("Checking URLs..."),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false))

	// Goroutine to check URLs
	for i := 0; i < 10; i++ { // Number of concurrent workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlsChan {
				currentProcessed := atomic.AddUint64(&totalProcessed, 1)
				client = clients[rand.Intn(len(clients))]
				if checkURL(url, client) {
					writer.WriteString(url + "\n")
					writer.Flush()
					atomic.AddUint64(&liveCount, 1)
				}
				bar.Describe(fmt.Sprintf("Checking URLs... (%d/%d)", currentProcessed, totalURLs))
				bar.Add(1)
			}
		}()
	}

	// Send URLs to the channel
	go func() {
		for _, url := range urls {
			urlsChan <- url
		}
		close(urlsChan)
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Printf("\nTotal URLs processed: %d\n", totalProcessed)
	fmt.Printf("Live URLs found: %d\n", liveCount)

	fmt.Printf("Live URLs have been written to %s\n", outputFile)
}
