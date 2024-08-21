# Porkbun Go SDK

[![Go Report Card](https://goreportcard.com/badge/github.com/tuzzmaniandevil/porkbun)](https://goreportcard.com/report/github.com/tuzzmaniandevil/porkbun)
[![GoDoc](https://godoc.org/github.com/tuzzmaniandevil/porkbun?status.svg)](https://godoc.org/github.com/tuzzmaniandevil/porkbun)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The Porkbun Go SDK is a fully-featured Go client for interacting with the [Porkbun API v3](https://porkbun.com/api/json/v3/documentation). This SDK simplifies integration with Porkbun's services, providing an easy-to-use and comprehensive interface for managing domains, DNS records, SSL certificates, and more.

## Features

- Domain management: List, create, update, and delete domains.
- DNS management: Full control over DNS records including creation, retrieval, updating, and deletion.
- SSL management: Retrieve SSL certificate bundles for domains.
- URL forwarding: Manage domain URL forwarding settings.
- Built-in error handling and support for custom data types (e.g., BoolString, BoolNumber).

## Installation

Install the SDK using `go get`:

```bash
go get github.com/tuzzmaniandevil/porkbun
```

## Usage

### Basic Usage

Below is a quick example demonstrating how to use the SDK to list all domains in your Porkbun account:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tuzzmaniandevil/porkbun"
)

func main() {
    client := porkbun.NewClient(&porkbun.Options{
        ApiKey:       "your_api_key",
        SecretApiKey: "your_secret_api_key",
    })

    resp, err := client.Domains.ListDomains(context.Background(), &porkbun.DomainListOptions{})
    if err != nil {
        log.Fatalf("Error listing domains: %v", err)
    }

    for _, domain := range resp.Domains {
        fmt.Println(domain.Domain)
    }
}
```

### Advanced Usage

For advanced usage, including custom API requests and handling more complex scenarios, refer to the [examples](https://github.com/tuzzmaniandevil/porkbun/tree/main/examples) directory in the repository.

## Documentation

Comprehensive documentation is available on [GoDoc](https://godoc.org/github.com/tuzzmaniandevil/porkbun).

## Testing

Run the tests using `go test`:

```bash
go test ./...
```

Ensure you have your API credentials set up in the environment or replace them directly in the test files for local testing.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request. For major changes, please open an issue first to discuss what you would like to change.

### Guidelines

- Write clear, concise commit messages.
- Ensure all tests pass before submitting a pull request.
- Follow the existing code style and format your code with `gofmt`.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [Porkbun](https://porkbun.com) for providing a robust API.
- [Go](https://golang.org) community for tools and inspiration.
