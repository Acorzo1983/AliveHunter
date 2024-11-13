
# AliveHunter

AliveHunter is a tool written in Go to check if URLs are alive (respond with HTTP 200 OK). It is designed to work with subdomain lists and supports the use of proxies to distribute requests.

## Features

- URL verification in blocks for improved efficiency.
- Supports multiple retries and configurable timeout.
- Supports proxies to distribute the load of requests.
- Detailed error logging in `error_log.txt`.
- Progress bar to visualize the status of the process.

## Requirements

- Go 1.16 or higher
- The following Go libraries:
  - `github.com/fatih/color`
  - `github.com/schollz/progressbar/v3`

To install the dependencies, run:

```sh
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
```

## Installation

To install `AliveHunter`, download the `install.sh` file, make it executable, and run it:

### OneLiner
```sh
git clone https://github.com/Acorzo1983/AliveHunter.git && cd AliveHunter && chmod +x install.sh && ./install.sh
```
### Manual Install
```sh
git clone https://github.com/Acorzo1983/AliveHunter.git
```
```sh
cd AliveHunter
```
```sh
chmod +x install.sh
```
```sh
./install.sh
```

## Usage

### Usage Examples

1. **Basic Usage**:

```sh
go run AliveHunter.go -l subdomainlist.txt
```

2. **Save results to a specific file**:

```sh
go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
```

3. **Use with a list of proxies**:

```sh
go run AliveHunter.go -l subdomainlist.txt -p proxylist.txt
```

4. **Configure the number of retries and timeout**:

```sh
go run AliveHunter.go -l subdomainlist.txt -r 5 -t 15
```

5. **Divide URLs into a maximum number of blocks**:

```sh
go run AliveHunter.go -l subdomainlist.txt -b 100
```

6. **Check only HTTPS URLs**:

```sh
go run AliveHunter.go -l subdomainlist.txt --https
```

### Complete Example with Subfinder

To use `AliveHunter` together with `subfinder`:

1. Generate the subdomains file with `subfinder`:

```sh
subfinder -d example.com --silent -o subdomainlist.txt
```

2. Run `AliveHunter` with the generated file:

```sh
go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
```

### Use of Proxychains

To use `proxychains` for multiproxying:

```sh
proxychains go run AliveHunter.go -l subdomainlist.txt
```

### Help

To display the help message:

```sh
go run AliveHunter.go -h
```

## Options

```
-l string
      File containing URLs to check (required)
-o string
      Output file to save the results (optional, default is <input_file>_alive.txt)
-p string
      File containing proxy list (optional)
-r int
      Number of retries for failed requests (default 5)
-t int
      Timeout for HTTP requests in seconds (default 15)
-b int
      Maximum number of blocks to divide (default 1000)
--https
      Check only HTTPS URLs
-h    Show help message
```

## Made with ❤️ by Albert.C
