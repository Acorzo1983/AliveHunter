package main

import (
    "bufio"
    "context"
    "flag"
    "fmt"
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
    "golang.org/x/time/rate"
)

var (
    startTime    time.Time
    lastUpdate   time.Time
    lastFoundURL atomic.Value
    urlsChecked  uint64
    liveCount    uint64
)

type Result struct {
    URL   string
    Alive bool
}

func updateProgress(currentURL string) {
    now := time.Now()
    if now.Sub(lastUpdate) >= time.Millisecond*100 {
        lastUpdate = now
        elapsed := now.Sub(startTime).Seconds()
        speed := float64(atomic.LoadUint64(&urlsChecked)) / elapsed
        lastFound := lastFoundURL.Load()
        
        // Clear previous lines and move cursor up
        fmt.Print("\033[2K\r")
        fmt.Print("\033[A\033[2K\r")
        fmt.Print("\033[A\033[2K\r")
        fmt.Print("\033[A\033[2K\r")
        
        // Print status
        fmt.Printf("Checking: %s\n", currentURL)
        if lastFound != nil {
            fmt.Printf("Last Found: %s\n", lastFound.(string))
        }
        fmt.Printf("Progress: %d found - %.1f URLs/sec\n", atomic.LoadUint64(&liveCount), speed)
        
        // Print progress bar
        width := 50
        progress := float64(atomic.LoadUint64(&urlsChecked)) / float64(totalURLs)
        completed := int(progress * float64(width))
        fmt.Printf("[%s%s] %.0f%%\n",
            strings.Repeat("=", completed),
            strings.Repeat("-", width-completed),
            progress*100)
    }
}

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
            url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
            urls = append(urls, url)
        }
    }
    return urls, scanner.Err()
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
        ForceAttemptHTTP2:  true,
    }
    
    return &http.Client{
        Transport: transport,
        Timeout:   15 * time.Second,
    }, nil
}

func checkURL(ctx context.Context, baseURL string, client *http.Client, limiter *rate.Limiter, httpsOnly bool) (string, bool) {
    if err := limiter.Wait(ctx); err != nil {
        return "", false
    }

    protocols := []string{"https://"}
    if !httpsOnly {
        protocols = append(protocols, "http://")
    }

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

var totalURLs int

func processURLs(ctx context.Context, urls []string, results chan<- Result, client *http.Client, limiter *rate.Limiter, httpsOnly bool) {
    for _, u := range urls {
        select {
        case <-ctx.Done():
            return
        default:
            fullURL, alive := checkURL(ctx, u, client, limiter, httpsOnly)
            if alive {
                lastFoundURL.Store(fullURL)
                atomic.AddUint64(&liveCount, 1)
            }
            atomic.AddUint64(&urlsChecked, 1)
            results <- Result{URL: fullURL, Alive: alive}
            updateProgress(u)
        }
    }
}

func main() {
    startTime = time.Now()
    lastUpdate = startTime
    
    color.New(color.FgHiCyan).Println("AliveHunter v1.4")
    color.New(color.FgHiYellow).Println("Made with love by Albert.C")
    fmt.Println()

    inputFile := flag.String("l", "", "File containing URLs to check")
    outputFile := flag.String("o", "", "Output file for results")
    proxyFile := flag.String("p", "", "File containing proxy list")
    workers := flag.Int("w", 10, "Number of concurrent workers")
    rateLimit := flag.Float64("rate", 10, "Requests per second")
    httpsOnly := flag.Bool("https", false, "Check only HTTPS URLs")
    flag.Parse()

    if *inputFile == "" {
        fmt.Println("Please specify an input file using -l flag")
        return
    }

    urls, err := readURLsFromFile(*inputFile)
    if err != nil {
        fmt.Printf("Error reading URLs: %v\n", err)
        return
    }

    totalURLs = len(urls)
    if totalURLs == 0 {
        fmt.Println("No URLs found in input file")
        return
    }

    outFileName := *outputFile
    if outFileName == "" {
        outFileName = strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile)) + "_alive.txt"
    }

    outFile, err := os.Create(outFileName)
    if err != nil {
        fmt.Printf("Error creating output file: %v\n", err)
        return
    }
    defer outFile.Close()

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
                DisableKeepAlives:  true,
                MaxIdleConns:       100,
                IdleConnTimeout:    90 * time.Second,
                ForceAttemptHTTP2:  true,
            },
        })
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()

    limiter := rate.NewLimiter(rate.Limit(*rateLimit), 1)
    results := make(chan Result, *workers)
    var wg sync.WaitGroup

    urlsPerWorker := (totalURLs + *workers - 1) / *workers
    for i := 0; i < *workers; i++ {
        start := i * urlsPerWorker
        end := start + urlsPerWorker
        if end > totalURLs {
            end = totalURLs
        }
        if start >= end {
            break
        }

        wg.Add(1)
        go func(workerURLs []string) {
            defer wg.Done()
            client := clients[0]
            if len(clients) > 1 {
                client = clients[i%len(clients)]
            }
            processURLs(ctx, workerURLs, results, client, limiter, *httpsOnly)
        }(urls[start:end])
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    writer := bufio.NewWriter(outFile)
    for result := range results {
        if result.Alive {
            writer.WriteString(result.URL + "\n")
            writer.Flush()
        }
    }

    // Clear progress display
    fmt.Print("\033[2K\r")
    fmt.Print("\033[A\033[2K\r")
    fmt.Print("\033[A\033[2K\r")
    fmt.Print("\033[A\033[2K\r")

    fmt.Printf("\nTotal URLs processed: %d\n", totalURLs)
    fmt.Printf("Live URLs found: %d\n", atomic.LoadUint64(&liveCount))
    fmt.Printf("Results saved to: %s\n", outFileName)
    fmt.Printf("Total execution time: %s\n", time.Since(startTime))
}
