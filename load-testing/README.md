# Load Testing

## WARNING

**DISCLAIMER:** Use only on resources YOU OWN or HAVE EXPLICIT WRITTEN PERMISSION to test. This is why we have put `localhost` as a placeholder in the code. Replace it ONLY with IP addresses, domains, etc. that you have approval on! You agree to not hold us responsible or liable for any misuse by you or others.

## Features

- Configuration file support for defining multiple requests
- Automatic request cycling to simulate diverse traffic patterns
- Custom headers per request

## Requirements

- Golang v1.24.x

## Usage

```bash
# Build the application
go build ./cmd/load-test/

# Run with a configuration file (required)
./load-test -config ./example-config.json
```

### Test Parameters

All test parameters are optional. If not specified, default values will be used:

- `duration`: Test duration in seconds (default: 10)
- `rate`: Number of requests per second (default: 150)
- `timeout`: Request timeout in seconds (default: 5)
- `warmupDelay`: Delay before starting the test in seconds (default: 15)
- `keepAlive`: Whether to keep HTTP connections alive (default: false)
- `http2`: Whether to use HTTP/2 (default: false)
- `redirects`: Maximum number of redirects to follow (default: 0)

### Request Configuration Options

- `method`: HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
- `url`: Target URL for the request
- `body`: Request body (optional, for POST/PUT/PATCH requests)
- `contentType`: Content-Type header (optional, e.g., "application/json", "text/plain")
- `headers`: Additional custom headers (optional)

## Uses Vegeta Library

- [Vegeta GitHub](https://github.com/tsenart/vegeta)
- [Vegeta GoDoc](https://pkg.go.dev/github.com/tsenart/vegeta/lib)

## Resources and Machine Types

This process is generall bottlenecked by CPU and Network.  
You should be fine with ~4GiB of RAM.  
Add as many CPUs as possible (and maybe NICs).  
ALWAYS use a wired connection when able.

## Alternatives

The performance and simplicity of `vegeta` has been impressive and using it is recommended.  
However, if you want to look for alternatives, look at some options below.

- [JMeter](https://jmeter.apache.org/)
- [Gatling](https://github.com/gatling/gatling) also has a [SaaS Hosted Version](https://gatling.io/)
- [Loader.io](https://loader.io/) (Commercial SaaS, Freemium)
