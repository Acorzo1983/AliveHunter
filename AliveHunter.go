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
	"golang.org/x/time/rate"
)

type Result struct {
	URL   string
	Alive bool
}

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
			// Remove any protocol prefix if present
			url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
			urls = append(urls, url)
			count++
		}
	}
	return urls, count, scanner.Err()
}

func divideIntoBlocks(urls []string, numBlocks int) [][]string {
	if numBlocks <= 0 {
		numBlocks = 1
	}
	blockSize := (len(urls) + numBlocks - 1) / numBlocks // Round up division
	blocks := make([][]string, 0, numBlocks)
	
	for i := 0; i < len(urls); i += blockSize {
		end := i + blockSize
		if end > len(urls) {
			end = len(urls)
		}
		blocks = append(blocks, urls[i:end])
	}
	return blocks
}

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
	return proxies, scanner.Err()
}

func createClientWithProxy(proxyURL string) (*http.Client, error) {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	
	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxy),
		DisableKeepAlives:  true,
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}, nil
}

func checkURL(ctx context.Context, baseURL string, client *http.Client, limiter *rate.Limiter) (string, bool) {
	if err := limiter.Wait(ctx); err != nil {
		return "", false
	}

	protocols := []string{"https://", "http://"}
	for _, protocol := range protocols {
		fullURL := protocol + baseURL
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			return fullURL, true
		}
	}
	
	return "", false
}

func processURLs(ctx context.Context, urls []string, results chan<- Result, client *http.Client, limiter *rate.Limiter, bar *progressbar.ProgressBar) {
	for _, u := range urls {
		select {
		case <-ctx.Done():
			return
		default:
			fullURL, alive := checkURL(ctx, u, client, limiter)
			results <- Result{URL: fullURL, Alive: alive}
			bar.Add(1)
		}
	}
}

func main() {
	startTime := time.Now()
	color.New(color.FgHiCyan).Println("AliveHunter v1.1")
	color.New(color.FgHiYellow).Println("Made with love by Albert.C")

	// Command line flags
	inputFile := flag.String("l", "", "File containing URLs to check (required)")
	outputFile := flag.String("o", "", "Output file for results (optional)")
	proxyFile := flag.String("p", "", "File containing proxy list (optional)")
	rateLimit := flag.Float64("rate", 10, "Requests per second")
	workers := flag.Int("w", 10, "Number of concurrent workers")
	help := flag.Bool("h", false, "Show help message")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *inputFile == "" {
		fmt.Println("Error: Input file is required. Use -h for help.")
		return
	}

	// Setup output file
	outFileName := *outputFile
	if outFileName == "" {
		outFileName = strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + "_alive.txt"
	}

	// Read URLs
	urls, totalURLs, err := readURLsFromFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading URLs: %v\n", err)
		return
	}

	if totalURLs == 0 {
		fmt.Println("No URLs found in input file.")
		return
	}

	// Setup clients
	var clients []*http.Client
	if *proxyFile != "" {
		proxies, err := readProxiesFromFile(*proxyFile)
		if err != nil {
			fmt.Printf("Error reading proxies: %v\n", err)
			return
		}
		for _, proxy := range proxies {
			if client, err := createClientWithProxy(proxy); err == nil {
				clients = append(clients, client)
			}
		}
	}
	if len(clients) == 0 {
		clients = append(clients, &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
				MaxIdleConns:      100,
				IdleConnTimeout:   90 * time.Second,
			},
		})
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Setup progress bar
	bar := progressbar.NewOptions(totalURLs,
		progressbar.OptionSetDescription("Checking URLs..."),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false),
	)

	// Setup rate limiter
	limiter := rate.NewLimiter(rate.Limit(*rateLimit), 1)

	// Create output file
	outFile, err := os.Create(outFileName)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outFile.Close()

	// Process URLs
	results := make(chan Result, *workers)
	var wg sync.WaitGroup
	blockSize := (totalURLs + *workers - 1) / *workers

	// Start workers
	for i := 0; i < totalURLs; i += blockSize {
		end := i + blockSize
		if end > totalURLs {
			end = totalURLs
		}

		wg.Add(1)
		go func(urls []string) {
			defer wg.Done()
			client := clients[rand.Intn(len(clients))]
			processURLs(ctx, urls, results, client, limiter, bar)
		}(urls[i:end])
	}

	// Start result writer
	var liveCount uint64
	go func() {
		writer := bufio.NewWriter(outFile)
		defer writer.Flush()

		for result := range results {
			if result.Alive {
				atomic.AddUint64(&liveCount, 1)
				writer.WriteString(result.URL + "\n")
				writer.Flush()
			}
		}
	}()

	// Wait for completion
	wg.Wait()
	close(results)

	// Print summary
	fmt.Printf("\nTotal URLs processed: %d\n", totalURLs)
	fmt.Printf("Live URLs found: %d\n", atomic.LoadUint64(&liveCount))
	fmt.Printf("Results saved to: %s\n", outFileName)
	fmt.Printf("Total execution time: %s\n", time.Since(startTime))
}

func printHelp() {
	fmt.Println(`AliveHunter Usage:
  -l string    Input file containing URLs (required)
  -o string    Output file for results (default: inputfile_alive.txt)
  -p string    Proxy list file (optional)
  -rate float  Requests per second (default: 10)
  -w int       Number of concurrent workers (default: 10)
  -h           Show this help message

Examples:
  ./alivehunter -l domains.txt
  ./alivehunter -l domains.txt -o results.txt -rate 20
  ./alivehunter -l domains.txt -p proxies.txt -w 20`)
}
