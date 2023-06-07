![Go version](https://img.shields.io/badge/Go-v1.19-blue.svg) [![Contribute](https://img.shields.io/badge/Contribute-Welcome-green.svg)](CONTRIBUTING.md)

# dnsgrab

This is a simple Go library and CLI tool that allows you to retrieve DNS servers from one or multiple hosts. When querying domains, the library incorporates a curated list of public resolvers from the [publicresolvers](https://github.com/root4loot/publicresolvers) library, expanding the available options for DNS resolution alongside the ability to provide your own custom resolvers using appropriate flags.

## Installation

### Go
```
go install github.com/root4loot/dnsgrab/cmd/dnsgrab@latest
```

### Docker
```
git clone https://github.com/root4loot/dnsgrab.git && cd dnsgrab
docker build -t dnsgrab .
docker run -it dnsgrab -h
```

## Usage
```
Usage: ./dnsgrab [options] (-h <host>|-i hosts.txt)

TARGETTING:
   -h,   --host          target host (comma separated)
   -i,   --infile        file containing hosts (newline separated)

CONFIGURATIONS:
   -r,  --resolvers      file containing list of resolvers   
   -c,  --concurrency    number of concurrent requests       (Default: 10 requests)
   -t,  --timeout        max request timeout                 (Default: 3 seconds)
   -d,  --delay          delay between requests              (Default: 0 milliseconds)
   -dj, --delay-jitter   max jitter between requests         (Default: 0 milliseconds)

OUTPUT:
   -o,  --outfile        output results to given file
   -s,  --silence        silence everything
   -v,  --verbose        verbose output
        --version        display version
```

## Examples
```
$ dnsgrab -h hackerone.com
104.16.99.52:53
```

```
$ dnsgrab -i hosts.txt
104.16.62.39:53
104.17.206.78:53
104.18.69.91:53
104.18.110.82:53
104.18.107.24:53
104.17.70.206:53
104.16.99.52:53
104.18.70.91:53
35.166.157.178:53
35.166.157.178:53
104.17.73.206:53
10.13.22.219:53
```


## Library
```
go get github.com/root4loot/dnsgrab@master
```

```go
package main

import (
	"fmt"

	"github.com/root4loot/dnsgrab"
)

func main() {
	single()
	multiple()
	multipleStream()
}

func single() {
	fmt.Println("Running against single host")
	results := dnsgrab.Single("hackerone.com")
	fmt.Println(results)
}

func multiple() {
	fmt.Println("Running against multiple hosts")
	results := dnsgrab.Multiple([]string{"hackerone.com", "bugcrowd.com", "intigriti.com"})
	fmt.Println(results)
}

func multipleStream() {
	fmt.Println("Running against multiple hosts (async)")
	targets := []string{"hackerone.com", "bugcrowd.com", "intigriti.com"}

	// initialize runner
	dnsgrab := dnsgrab.NewRunner()

	// OPTIONAL: set options
	// dnsgrab.Options.Resolvers = []string{""}
	// dnsgrab.Options.Concurrency = 0
	// dnsgrab.Options.Timeout = 0
	// dnsgrab.Options.Delay = 0
	// dnsgrab.Options.DelayJitter = 0
	// dnsgrab.Options.Verbose = false

	// process results
	go func() {
		for result := range dnsgrab.Results {
			fmt.Println(result.Host)
		}
	}()

	// run dnsgrab against targets
	dnsgrab.MultipleStream(targets...)
}
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
