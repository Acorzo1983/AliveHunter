
# AliveHunter v0.9

AliveHunter is a tool for checking the availability of URLs using HTTP/HTTPS, with support for proxies and `proxychains` for distributed scanning. Save results efficiently and track progress interactively.

## Features

- Checks the availability of URLs using HTTP and HTTPS.
- Supports proxies specified in a file.
- Interactive progress bar.
- Option to use `proxychains` for multi-node proxying.

## Installation

1. **Clone the repository**:

    ```bash
    git clone https://github.com/Acorzo1983/AliveHunter.git
    cd AliveHunter
    ```

2. **Initialize the Go module**:

    ```bash
    go mod init AliveHunter
    ```

3. **Install dependencies**:

    ```bash
    go get github.com/fatih/color
    go get github.com/schollz/progressbar/v3
    ```

4. **Run `go mod tidy`**:

    ```bash
    go mod tidy
    ```

## Usage

### Running the script

To run the script specifying the input file and the proxy file (optional):

```bash
go run AliveHunter.go -l domains.txt -p proxy.txt
```

### Viewing help

To view help:

```bash
go run AliveHunter.go -h
```

### Using `proxychains`

To run the script with `proxychains`:

```bash
proxychains go run AliveHunter.go -l domains.txt
```

## Example

Suppose you have a file `domains.txt` with a list of URLs and a file `proxy.txt` with a list of proxies.

```bash
go run AliveHunter.go -l domains.txt -p proxy.txt
```

The result will be saved in a file called `url_alive.txt`.

## Contributing

Contributions are welcome. Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/new-feature`).
3. Make your changes and commit them (`git commit -am 'Add new feature'`).
4. Push to the branch (`git push origin feature/new-feature`).
5. Open a Pull Request.

## Author

Made with ❤️ by Albert.C
