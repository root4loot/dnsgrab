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
	result := dnsgrab.Single("hackerone.com")
	fmt.Println("Query host", result.RequestedHost, "Resolver Address", result.ResolverAddress)
}

func multiple() {
	fmt.Println("Running against multiple hosts")
	results := dnsgrab.Multiple([]string{"hackerone.com", "bugcrowd.com", "intigriti.com"})
	for _, result := range results {
		fmt.Println("Query host", result.RequestedHost, "Resolver Address", result.ResolverAddress)
	}
}

func multipleStream() {
	fmt.Println("Running against multiple hosts (async)")
	targets := []string{"hackerone.com", "bugcasdasdrowd.com", "intigriti.com"}

	// initialize runner
	dnsgrab := dnsgrab.NewRunner()

	// OPTIONAL: set options
	// dnsgrab.Options.Resolvers = []string{""}
	// dnsgrab.Options.Concurrency = 0
	// dnsgrab.Options.Timeout = 0
	// dnsgrab.Options.Delay = 0
	// dnsgrab.Options.DelayJitter = 0
	dnsgrab.Options.Verbose = true

	// process results
	go func() {
		for result := range dnsgrab.Results {
			fmt.Println("Query host", result.RequestedHost, "Resolver Address", result.ResolverAddress)
		}
	}()

	// run dnsgrab against targets
	dnsgrab.MultipleStream(targets...)
}
