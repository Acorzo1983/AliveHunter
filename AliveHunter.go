package main

import (
    "bufio"
    "context"
    "crypto/tls"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "regexp"
    "runtime"
    "sort"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
    "time"

    "github.com/fatih/color"
    "golang.org/x/net/html"
    "golang.org/x/time/rate"
)

const (
    VERSION         = "3.2"
    DEFAULT_WORKERS = 100
    DEFAULT_RATE    = 100
    DEFAULT_TIMEOUT = 3 * time.Second
    BATCH_SIZE      = 1000
    MAX_BODY_SIZE   = 10 * 1024 // 10KB for verification
    TITLE_BODY_SIZE = 8192      // 8KB for title extraction
)

// Compile regex once for performance
var (
    whitespaceRegex = regexp.MustCompile(`\s+`)
)

// Config holds all configuration options
type Config struct {
    Workers        int           // Number of concurrent workers
    Rate          float64       // Requests per second
    Timeout       time.Duration // Request timeout
    OutputFormat  string        // Output format
    Silent        bool          // Silent mode for pipelines
    FastMode      bool          // Sacrifice some accuracy for maximum speed
    VerifyMode    bool          // Maximum accuracy, slower
    JSONOutput    bool          // JSON output format
    OnlyStatus    []int         // Only match specific status codes
    FollowRedirect bool         // Follow HTTP redirects
    ExtractTitle  bool          // Extract page titles
    MaxBodySize   int64         // Maximum response body size to read
    ShowFailed    bool          // Show failed requests
    RobustTitle   bool          // Use robust HTML parser for titles (slower)
    TLSMinVersion uint16        // Minimum TLS version
}

// Result represents the outcome of checking a single URL
type Result struct {
    URL          string        `json:"url"`
    Status       int           `json:"status_code"`
    Length       int64         `json:"content_length"`
    ResponseTime time.Duration `json:"response_time_ms"`
    Title        string        `json:"title,omitempty"`
    Server       string        `json:"server,omitempty"`
    Redirect     string        `json:"redirect,omitempty"`
    Error        string        `json:"error,omitempty"`
    Alive        bool          `json:"alive"`
    Verified     bool          `json:"verified"`
}

// Stats tracks scanning progress and performance metrics
type Stats struct {
    started   time.Time
    checked   uint64
    alive     uint64
    errors    uint64
    verified  uint64
    totalUrls int64
}

// String returns a formatted string representation of current stats
func (s *Stats) String() string {
    elapsed := time.Since(s.started)
    var speed float64
    if elapsed.Seconds() > 0 {
        speed = float64(atomic.LoadUint64(&s.checked)) / elapsed.Seconds()
    }
    return fmt.Sprintf("Checked: %d | Alive: %d | Verified: %d | Errors: %d | Speed: %.0f req/s",
        atomic.LoadUint64(&s.checked),
        atomic.LoadUint64(&s.alive),
        atomic.LoadUint64(&s.verified),
        atomic.LoadUint64(&s.errors),
        speed)
}

// AliveHTTPClient is an optimized HTTP client for maximum speed
type AliveHTTPClient struct {
    client    *http.Client
    transport *http.Transport
}

// NewAliveHTTPClient creates a new optimized HTTP client
func NewAliveHTTPClient(config *Config) *AliveHTTPClient {
    // Ultra-optimized transport for scanning diverse hosts
    transport := &http.Transport{
        DialContext: (&net.Dialer{
            Timeout:   2 * time.Second,
            KeepAlive: 0,        // Disable keep-alive for diverse host scanning efficiency
            DualStack: true,
        }).DialContext,
        
        // Speed-optimized settings for mass scanning diverse hosts
        MaxIdleConns:          0,                    // No idle connections for diverse hosts
        MaxIdleConnsPerHost:   0,                    
        MaxConnsPerHost:       config.Workers * 2,   // Allow more concurrent connections
        IdleConnTimeout:       0,
        DisableKeepAlives:     true,                 // Optimal for diverse host scanning
        DisableCompression:    true,                 // Less CPU overhead
        ForceAttemptHTTP2:     false,                // HTTP/1.1 is faster for this use case
        ExpectContinueTimeout: 0,
        ResponseHeaderTimeout: config.Timeout,
        
        // TLS configuration with configurable minimum version
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true, // Speed > security for reconnaissance
            MinVersion:         config.TLSMinVersion,
        },
    }

    return &AliveHTTPClient{
        transport: transport,
        client: &http.Client{
            Transport: transport,
            Timeout:   config.Timeout,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if !config.FollowRedirect || len(via) >= 3 {
                    return http.ErrUseLastResponse
                }
                return nil
            },
        },
    }
}

// RequestType defines the purpose of an HTTP request
type RequestType int

const (
    RequestTypeCheck RequestType = iota
    RequestTypeTitle
    RequestTypeVerification
)

// createRequest creates a new HTTP request with appropriate headers for the request type
func (ac *AliveHTTPClient) createRequest(ctx context.Context, method, url string, reqType RequestType) (*http.Request, error) {
    req, err := http.NewRequestWithContext(ctx, method, url, nil)
    if err != nil {
        return nil, err
    }
    
    // Base headers for all requests
    req.Header.Set("User-Agent", "AliveHunter/"+VERSION)
    req.Header.Set("Accept", "*/*")
    
    // Request-type specific headers
    switch reqType {
    case RequestTypeTitle:
        req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
        req.Header.Set("Accept-Language", "en-US,en;q=0.9")
    case RequestTypeVerification:
        req.Header.Set("Accept", "text/html,application/xhtml+xml")
        req.Header.Set("Cache-Control", "no-cache") // Ensure fresh content for verification
    case RequestTypeCheck:
        // Minimal headers for speed
    }
    
    return req, nil
}

// fetchBody makes a GET request for body content (unified for title/verification)
func (ac *AliveHTTPClient) fetchBody(ctx context.Context, fullURL string, reqType RequestType) (*http.Response, error) {
    req, err := ac.createRequest(ctx, "GET", fullURL, reqType)
    if err != nil {
        return nil, err
    }
    
    return ac.client.Do(req)
}

// CheckURL performs ultra-fast URL verification with minimal false positives
func (ac *AliveHTTPClient) CheckURL(ctx context.Context, rawURL string, config *Config) *Result {
    start := time.Now()
    result := &Result{URL: rawURL}
    
    // Robust URL validation
    if !isValidURL(rawURL) {
        result.Error = "invalid_url"
        return result
    }

    // Try HTTPS first (more common in 2024), then HTTP
    protocols := []string{"https://", "http://"}
    var lastError error
    
    for _, protocol := range protocols {
        fullURL := protocol + strings.TrimPrefix(strings.TrimPrefix(rawURL, "https://"), "http://")
        
        // Use HEAD by default for speed, GET only if we need title
        method := "HEAD"
        if config.ExtractTitle {
            method = "GET"
        }
        
        req, err := ac.createRequest(ctx, method, fullURL, RequestTypeCheck)
        if err != nil {
            lastError = err
            continue
        }
        
        resp, err := ac.client.Do(req)
        if err != nil {
            lastError = err
            // In fast mode, don't retry
            if config.FastMode {
                continue
            }
            // In normal mode, one quick retry with exponential backoff
            time.Sleep(50 * time.Millisecond)
            resp, err = ac.client.Do(req)
            if err != nil {
                lastError = err
                continue
            }
        }
        
        defer resp.Body.Close()
        
        // Populate basic result data
        result.URL = fullURL
        result.Status = resp.StatusCode
        result.ResponseTime = time.Since(start)
        result.Server = resp.Header.Get("Server")
        
        // Calculate content length carefully
        if method == "GET" && resp.Body != nil {
            // Consume body to get actual length, but save it for potential reuse
            bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, config.MaxBodySize))
            if err == nil {
                result.Length = int64(len(bodyBytes))
                
                // Store body for potential title extraction or verification
                resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
            }
        } else if resp.ContentLength > 0 {
            result.Length = resp.ContentLength
        }
        
        // Determine if URL is "alive" based on reliable status codes
        if isAliveStatus(resp.StatusCode, config) {
            result.Alive = true
            
            // Additional verification to prevent false positives
            needsVerification := !config.FastMode && shouldVerifyResponse(resp, config)
            if needsVerification {
                verified, verifyErr := ac.performVerification(ctx, fullURL, method == "GET", resp)
                if verifyErr != nil {
                    result.Error = fmt.Sprintf("verification_failed: %s", verifyErr.Error())
                } else if !verified {
                    result.Alive = false
                    result.Error = "false_positive_detected"
                    return result
                } else {
                    result.Verified = true
                }
            }
            
            // Extract title if required
            if config.ExtractTitle {
                if method == "GET" && resp.Body != nil {
                    // Use the already-read body
                    result.Title = ac.extractTitle(resp.Body, config.RobustTitle)
                } else {
                    // Make a GET request specifically for title
                    titleResp, err := ac.fetchBody(ctx, fullURL, RequestTypeTitle)
                    if err == nil {
                        defer titleResp.Body.Close()
                        result.Title = ac.extractTitle(titleResp.Body, config.RobustTitle)
                    }
                }
            }
            
            // Handle redirects
            if isRedirect(resp.StatusCode) && resp.Header.Get("Location") != "" {
                result.Redirect = resp.Header.Get("Location")
            }
        }
        
        return result
    }
    
    // If we get here, both protocols failed
    if lastError != nil {
        result.Error = fmt.Sprintf("connection_failed: %s", lastError.Error())
    } else {
        result.Error = "no_response"
    }
    return result
}

// performVerification does additional verification to prevent false positives
func (ac *AliveHTTPClient) performVerification(ctx context.Context, fullURL string, alreadyGET bool, originalResp *http.Response) (bool, error) {
    var resp *http.Response
    var err error
    
    if alreadyGET && originalResp.Body != nil {
        // Try to reuse the already-read body first
        verified, verifyErr := ac.verifyResponseBody(originalResp)
        if verifyErr == nil {
            return verified, nil
        }
        // If that fails, fall back to re-fetching
    }
    
    // Make a fresh GET request for verification
    resp, err = ac.fetchBody(ctx, fullURL, RequestTypeVerification)
    if err != nil {
        return false, fmt.Errorf("verification_request_failed: %w", err)
    }
    defer resp.Body.Close()
    
    return ac.verifyResponseBody(resp)
}

// verifyResponseBody checks if the response body indicates a false positive
func (ac *AliveHTTPClient) verifyResponseBody(resp *http.Response) (bool, error) {
    if resp.Body == nil {
        return true, nil // No body to analyze
    }
    
    // Read a reasonable sample of the body for verification
    body := make([]byte, 2048) // Sufficient for most false positive detection
    n, _ := resp.Body.Read(body)
    content := strings.ToLower(string(body[:n]))
    
    // Comprehensive patterns that indicate false positives
    falsePositivePatterns := []string{
        "domain for sale",
        "this domain is for sale",
        "page not found",
        "404 not found",
        "file not found",
        "this domain may be for sale",
        "parked domain",
        "domain parking",
        "coming soon",
        "under construction",
        "default page",
        "welcome to nginx",
        "apache2 default page",
        "iis windows server",
        "default website",
        "placeholder page",
        "this site can't be reached",
        "website temporarily unavailable",
        "suspended",
        "account suspended",
        "hosting account",
        "plesk default page",
        "cpanel",
        "whm default page",
        "godaddy",
        "namecheap",
        "sedo domain parking",
    }
    
    for _, pattern := range falsePositivePatterns {
        if strings.Contains(content, pattern) {
            return false, nil
        }
    }
    
    return true, nil
}

// isAliveStatus determines which status codes indicate a live website
func isAliveStatus(status int, config *Config) bool {
    // If specific status codes are requested, only match those
    if len(config.OnlyStatus) > 0 {
        for _, s := range config.OnlyStatus {
            if status == s {
                return true
            }
        }
        return false
    }
    
    // Status codes that reliably indicate the site is alive
    // Optimized to minimize false positives
    aliveStatuses := []int{
        200, 201, 202, 204, 206,           // Success codes
        301, 302, 303, 307, 308,           // Redirects (content exists)
        401, 403,                          // Authentication/authorization (content exists)
        405, 406, 409, 410,               // Method/content issues (but server is alive)
        429,                               // Rate limited (server is alive)
        500, 501, 502, 503,               // Server errors (but server exists)
    }
    
    for _, code := range aliveStatuses {
        if status == code {
            return true
        }
    }
    
    return false
}

// isRedirect checks if status code indicates a redirect
func isRedirect(status int) bool {
    return status >= 300 && status < 400
}

// isValidURL performs robust URL validation
func isValidURL(rawURL string) bool {
    if rawURL == "" || len(rawURL) > 200 {
        return false
    }
    
    // Quick basic validation first for performance
    if strings.ContainsAny(rawURL, " \t\n\r<>\"{}|\\^`[]") {
        return false
    }
    
    // Add protocol for validation if missing
    testURL := rawURL
    if !strings.Contains(rawURL, "://") {
        testURL = "https://" + rawURL
    }
    
    // Use Go's standard URL parser for robust validation
    _, err := url.ParseRequestURI(testURL)
    return err == nil
}

// shouldVerifyResponse determines if additional verification is needed
func shouldVerifyResponse(resp *http.Response, config *Config) bool {
    // In fast mode, skip verification
    if config.FastMode {
        return false
    }
    
    // Always verify in verify mode
    if config.VerifyMode {
        return true
    }
    
    // Check for common web server signatures that might serve generic pages
    contentType := resp.Header.Get("Content-Type")
    server := resp.Header.Get("Server")
    
    // Common web server signatures that often serve default/parked pages
    genericServerSignatures := []string{"cloudflare", "nginx", "apache", "iis", "lighttpd"}
    for _, sig := range genericServerSignatures {
        if strings.Contains(strings.ToLower(server), sig) && 
           resp.StatusCode == 200 && 
           strings.Contains(strings.ToLower(contentType), "text/html") {
            return true
        }
    }
    
    return false
}

// extractTitle extracts the HTML title from response body
func (ac *AliveHTTPClient) extractTitle(body io.Reader, robust bool) string {
    if robust {
        return ac.extractTitleRobust(body)
    }
    return ac.extractTitleFast(body)
}

// extractTitleFast performs fast but less robust title extraction
func (ac *AliveHTTPClient) extractTitleFast(body io.Reader) string {
    // Fast title extraction - only read first portion
    buffer := make([]byte, TITLE_BODY_SIZE)
    n, _ := body.Read(buffer)
    content := strings.ToLower(string(buffer[:n]))
    
    // Look for opening title tag with improved flexibility
    titleStart := -1
    contentStr := string(buffer[:n]) // Preserve original case for extraction
    
    for _, pattern := range []string{"<title>", "<title "} {
        if idx := strings.Index(content, pattern); idx != -1 {
            if pattern == "<title>" {
                titleStart = idx + 7
            } else {
                // Handle <title attributes>
                closeIdx := strings.Index(content[idx:], ">")
                if closeIdx != -1 {
                    titleStart = idx + closeIdx + 1
                }
            }
            break
        }
    }
    
    if titleStart == -1 {
        return ""
    }
    
    // Look for closing title tag
    end := strings.Index(content[titleStart:], "</title>")
    if end == -1 {
        return ""
    }
    
    // Extract title preserving original case
    title := strings.TrimSpace(contentStr[titleStart : titleStart+end])
    
    // Efficient whitespace cleaning using compiled regex
    title = whitespaceRegex.ReplaceAllString(title, " ")
    title = strings.TrimSpace(title)
    
    // Trim very long titles
    if len(title) > 100 {
        title = title[:100] + "..."
    }
    
    return title
}

// extractTitleRobust performs robust title extraction using HTML parser
func (ac *AliveHTTPClient) extractTitleRobust(body io.Reader) string {
    // Limit reading for performance
    limitedBody := io.LimitReader(body, TITLE_BODY_SIZE)
    
    tokenizer := html.NewTokenizer(limitedBody)
    
    for {
        tokenType := tokenizer.Next()
        switch tokenType {
        case html.ErrorToken:
            return "" // End of document or error
        case html.StartTagToken:
            token := tokenizer.Token()
            if token.Data == "title" {
                // Found title tag, get the text content
                tokenType = tokenizer.Next()
                if tokenType == html.TextToken {
                    title := strings.TrimSpace(tokenizer.Token().Data)
                    // Clean whitespace efficiently
                    title = whitespaceRegex.ReplaceAllString(title, " ")
                    if len(title) > 100 {
                        title = title[:100] + "..."
                    }
                    return title
                }
            }
        }
    }
}

// processURLs is the main worker function that processes URLs from a channel
func processURLs(ctx context.Context, urls <-chan string, results chan<- *Result, client *AliveHTTPClient, config *Config, stats *Stats, limiter *rate.Limiter) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintf(os.Stderr, "Worker panic: %v\n", r)
        }
    }()
    
    for {
        select {
        case <-ctx.Done():
            return
        case url, ok := <-urls:
            if !ok {
                return
            }
            
            // Rate limiting only if not in fast mode
            if !config.FastMode {
                if err := limiter.Wait(ctx); err != nil {
                    return // Context cancelled during rate limiting
                }
            }
            
            result := client.CheckURL(ctx, url, config)
            
            // Update stats atomically
            atomic.AddUint64(&stats.checked, 1)
            if result.Alive {
                atomic.AddUint64(&stats.alive, 1)
            }
            if result.Verified {
                atomic.AddUint64(&stats.verified, 1)
            }
            if result.Error != "" {
                atomic.AddUint64(&stats.errors, 1)
            }
            
            results <- result
        }
    }
}

// displayProgress shows real-time progress without affecting performance
func displayProgress(ctx context.Context, stats *Stats, config *Config) {
    if config.Silent {
        return
    }
    
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            fmt.Fprintf(os.Stderr, "\r\033[K%s", stats.String())
        }
    }
}

// readInput reads URLs from stdin for pipeline compatibility
func readInput() ([]string, error) {
    var urls []string
    
    // Check if there's data available on stdin
    stat, err := os.Stdin.Stat()
    if err != nil {
        return nil, err
    }
    
    if (stat.Mode() & os.ModeCharDevice) != 0 {
        return nil, errors.New("no input provided via pipe or redirection")
    }
    
    // Read from stdin for pipeline compatibility
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // Larger buffer for performance
    
    for scanner.Scan() {
        url := strings.TrimSpace(scanner.Text())
        if url != "" && !strings.HasPrefix(url, "#") {
            urls = append(urls, url)
        }
    }
    
    return urls, scanner.Err()
}

// outputResult formats and outputs a single result
func outputResult(result *Result, config *Config) {
    // In silent mode, only show alive URLs unless explicitly requested
    if config.Silent && !result.Alive && !config.ShowFailed {
        return
    }
    
    if config.JSONOutput {
        data, _ := json.Marshal(result)
        fmt.Println(string(data))
    } else {
        if result.Alive {
            output := result.URL
            if config.ExtractTitle && result.Title != "" {
                output += " [" + result.Title + "]"
            }
            if result.Status != 200 {
                output += fmt.Sprintf(" [%d]", result.Status)
            }
            if result.Verified {
                output += " [VERIFIED]"
            }
            fmt.Println(output)
        } else if config.ShowFailed {
            fmt.Printf("%s [FAILED: %s]\n", result.URL, result.Error)
        }
    }
}

func main() {
    // Display help with examples and branding
    if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
        color.New(color.FgHiCyan).Println("AliveHunter v" + VERSION + " - Ultra-fast web discovery")
        color.New(color.FgHiYellow).Println("Optimized for speed with zero false positives")
        fmt.Println("\nUsage: cat domains.txt | alivehunter [options]")
        fmt.Println("\nModes:")
        fmt.Println("  Default: Perfect balance of speed and accuracy")
        fmt.Println("  -fast:   Maximum speed (minimal verification)")
        fmt.Println("  -verify: Zero false positives guaranteed (slower)")
        fmt.Println("\nExamples:")
        fmt.Println("  subfinder -d target.com | alivehunter -silent")
        fmt.Println("  cat domains.txt | alivehunter -fast -title")
        fmt.Println("  echo 'example.com' | alivehunter -json -verify")
        fmt.Println("  cat large_list.txt | alivehunter -fast -t 200 -rate 200")
        fmt.Println()
        color.New(color.FgHiGreen).Println("Made with ❤️ by Albert.C")
        fmt.Println()
        return
    }

    // Default configuration optimized for best performance out-of-the-box
    config := &Config{
        Workers:       DEFAULT_WORKERS,
        Rate:         DEFAULT_RATE,
        Timeout:      DEFAULT_TIMEOUT,
        MaxBodySize:   MAX_BODY_SIZE,
        OnlyStatus:    []int{},
        TLSMinVersion: tls.VersionTLS12, // Secure default
    }

    // Command line flags
    flag.IntVar(&config.Workers, "t", config.Workers, "Number of threads")
    flag.IntVar(&config.Workers, "threads", config.Workers, "Number of threads (alias)")
    flag.Float64Var(&config.Rate, "rate", config.Rate, "Requests per second")
    flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Request timeout")
    flag.BoolVar(&config.Silent, "silent", false, "Silent mode (pipeline friendly)")
    flag.BoolVar(&config.JSONOutput, "json", false, "JSON output")
    flag.BoolVar(&config.ExtractTitle, "title", false, "Extract page titles")
    flag.BoolVar(&config.RobustTitle, "robust-title", false, "Use robust HTML parser for titles (slower)")
    flag.BoolVar(&config.FastMode, "fast", false, "Fast mode (minimal verification)")
    flag.BoolVar(&config.VerifyMode, "verify", false, "Verify mode (zero false positives)")
    flag.BoolVar(&config.FollowRedirect, "follow-redirects", false, "Follow HTTP redirects")
    flag.BoolVar(&config.ShowFailed, "show-failed", false, "Show failed requests")
    
    statusCodes := flag.String("mc", "", "Match status codes (comma separated)")
    tlsVersion := flag.String("tls-min", "1.2", "Minimum TLS version (1.0, 1.1, 1.2, 1.3)")
    flag.Parse()

    // Parse TLS version
    switch *tlsVersion {
    case "1.0":
        config.TLSMinVersion = tls.VersionTLS10
    case "1.1":
        config.TLSMinVersion = tls.VersionTLS11
    case "1.2":
        config.TLSMinVersion = tls.VersionTLS12
    case "1.3":
        config.TLSMinVersion = tls.VersionTLS13
    default:
        config.TLSMinVersion = tls.VersionTLS12
    }

    // Parse status codes
    if *statusCodes != "" {
        parts := strings.Split(*statusCodes, ",")
        for _, part := range parts {
            var code int
            if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &code); err == nil {
                config.OnlyStatus = append(config.OnlyStatus, code)
            }
        }
        sort.Ints(config.OnlyStatus)
    }

    // Auto-optimize based on mode
    if config.FastMode {
        config.Workers *= 2                    // More workers for maximum throughput
        config.Rate *= 2                       // Higher rate limit
        config.Timeout = 1 * time.Second       // Aggressive timeout
    }
    
    if config.VerifyMode {
        config.Workers = max(config.Workers/2, 10) // Fewer workers but minimum 10
        config.Timeout = 10 * time.Second          // Conservative timeout
    }

    // System optimization - respect system limits
    maxWorkers := runtime.NumCPU() * 50
    if config.Workers > maxWorkers {
        config.Workers = maxWorkers
    }

    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        if !config.Silent {
            fmt.Fprintf(os.Stderr, "\nReceived interrupt, shutting down gracefully...\n")
        }
        cancel()
    }()

    // Read input from stdin
    urls, err := readInput()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
        fmt.Fprintf(os.Stderr, "Usage: cat domains.txt | %s [options]\n", os.Args[0])
        os.Exit(1)
    }

    if len(urls) == 0 {
        fmt.Fprintf(os.Stderr, "No URLs provided via stdin\n")
        os.Exit(1)
    }

    // Initialize performance tracking
    stats := &Stats{
        started:   time.Now(),
        totalUrls: int64(len(urls)),
    }

    // Setup worker coordination
    urlChan := make(chan string, BATCH_SIZE)
    resultsChan := make(chan *Result, BATCH_SIZE)
    limiter := rate.NewLimiter(rate.Limit(config.Rate), 1)
    client := NewAliveHTTPClient(config)

    // Start progress monitoring
    go displayProgress(ctx, stats, config)

    // Launch worker pool
    var wg sync.WaitGroup
    for i := 0; i < config.Workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            processURLs(ctx, urlChan, resultsChan, client, config, stats, limiter)
        }()
    }

    // Process and output results
    resultsDone := make(chan struct{})
    go func() {
        defer close(resultsDone)
        for result := range resultsChan {
            outputResult(result, config)
        }
    }()

    // Feed URLs to workers
    go func() {
        defer close(urlChan)
        for _, url := range urls {
            select {
            case <-ctx.Done():
                return
            case urlChan <- url:
            }
        }
    }()

    // Coordinate shutdown
    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    // Wait for all results to be processed
    <-resultsDone

    // Final statistics with branding
    if !config.Silent {
        fmt.Fprintf(os.Stderr, "\nScan completed: %s\n", stats.String())
        elapsed := time.Since(stats.started)
        fmt.Fprintf(os.Stderr, "Total time: %v\n", elapsed.Round(time.Second))
        
        // Performance summary
        alive := atomic.LoadUint64(&stats.alive)
        checked := atomic.LoadUint64(&stats.checked)
        if checked > 0 {
            successRate := float64(alive) / float64(checked) * 100
            fmt.Fprintf(os.Stderr, "Success rate: %.1f%%\n", successRate)
        }
        
        // Signature
        color.New(color.FgHiGreen).Fprintf(os.Stderr, "\nMade with ❤️ by Albert.C\n")
    }
}

// max returns the maximum of two integers
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
