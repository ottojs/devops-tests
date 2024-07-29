# Load Testing

## WARNING

**DISCLAIMER:** Use only on resources YOU OWN or HAVE EXPLICIT WRITTEN PERMISSION to test. This is why we have put `localhost` as a placeholder in the code. Replace it ONLY with IP addresses, domains, etc. that you have approval on! You agree to not hold us responsible or liable for any misuse by you or others.

## Requirements

- Golang v1.22.x

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
- [Loader.io](https://loader.io/) (Commercial Saas, Freemium)
