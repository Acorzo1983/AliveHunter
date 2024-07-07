# AliveHunter v1.0

AliveHunter is a tool for checking the availability of a list of URLs. Developed with love by Albert.C.

## Usage

```
go run AliveHunter.go -l subdomainlist.txt [-o output.txt] [-p proxylist.txt] [-r retries] [-t timeout] [-b maxBlocks] [-c concurrency] [--https]
```

### Options

- `-l string`: File containing URLs to check (use `-` to read from stdin) (required).
- `-o string`: Output file to save the results (optional, default is `<input_file>_alive.txt`).
- `-p string`: File containing proxy list (optional).
- `-r int`: Number of retries for failed requests (default 2).
- `-t int`: Timeout for HTTP requests in seconds (default 5).
- `-b int`: Maximum number of blocks to divide (default 1000).
- `-c int`: Number of concurrent workers (default 10).
- `--https`: Check only HTTPS URLs.
- `-h`: Show help message.

### Examples

```bash
subfinder -d example.com --silent -o subdomainlist.txt && go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
subfinder -d example.com --silent | go run AliveHunter.go -l - -o alive_subdomains.txt
go run AliveHunter.go -l subdomainlist.txt
go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
go run AliveHunter.go -l subdomainlist.txt -p proxylist.txt
go run AliveHunter.go -l subdomainlist.txt -r 5
go run AliveHunter.go -l subdomainlist.txt -t 15
go run AliveHunter.go -l subdomainlist.txt -b 100
go run AliveHunter.go -l subdomainlist.txt --https
```

### Installing Dependencies

Make sure to install the necessary dependencies with:

```bash
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
```

Or use the provided installer script:

```bash
chmod +x installer.sh
./installer.sh
```

### Using proxychains

You can also use proxychains for multi-node proxying:

```bash
proxychains go run AliveHunter.go -l subdomainlist.txt
```

## Contributing

Contributions are welcome. Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/new-feature`).
3. Make your changes and commit them (`git commit -am 'Add new feature'`).
4. Push to the branch (`git push origin feature/new-feature`).
5. Open a Pull Request.

## Author

Made with ❤️ by Albert.C
