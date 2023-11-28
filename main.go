package dnsgrab

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/root4loot/goutils/log"
	"github.com/root4loot/publicresolvers"
)

type Runner struct {
	Options Options     // options for the runner
	Results chan Result // channel to receive results
	Visited map[string]bool
}

// Options contains options for the runner
type Options struct {
	Concurrency int  // number of concurrent requests
	Timeout     int  // timeout in seconds
	Delay       int  // delay in seconds
	DelayJitter int  // delay jitter in seconds
	Verbose     bool // verbose logging
	Resolvers   []string
}

type Result struct {
	Host string
}

func init() {
	log.Init("dnsgrab")
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	publicresolvers, _ := publicresolvers.FetchResolversTrustedWithPort()

	return &Options{
		Concurrency: 10,
		Timeout:     3,
		Delay:       0,
		DelayJitter: 0,
		Resolvers:   publicresolvers,
		Verbose:     false,
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	options := DefaultOptions()
	if options.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	return &Runner{
		Results: make(chan Result),
		Visited: make(map[string]bool),
		Options: *options,
	}
}

// Single runs dnsgrab against a single host
func Single(host string) (result Result) {
	r := NewRunner()
	return r.worker(host)
}

// Multiple runs dnsgrab against multiple hosts with concurrency
func Multiple(hosts []string) (results []Result) {
	r := NewRunner()
	concurrency := r.Options.Concurrency
	if concurrency > len(hosts) {
		concurrency = len(hosts)
	}

	resultsChan := make(chan Result, len(hosts))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore
		go func(h string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore
			resultsChan <- Single(h)
			time.Sleep(time.Millisecond * 100) // make room for processing results
		}(host)
	}

	wg.Wait()
	close(resultsChan)

	for result := range resultsChan {
		results = append(results, result)
	}
	return
}

// MultipleStream runs dnsgrab against multiple hosts in async mode
func (r *Runner) MultipleStream(hosts ...string) {
	defer close(r.Results)

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup

	for _, host := range hosts {
		if !r.Visited[host] {
			r.Visited[host] = true

			sem <- struct{}{}
			wg.Add(1)
			go func(h string) {
				defer func() { <-sem }()
				defer wg.Done()
				r.Results <- Single(h)
				time.Sleep(time.Millisecond * 100) // make room for processing results
			}(host)
			time.Sleep(r.getDelay() * time.Millisecond) // delay between requests
		}
	}
	wg.Wait()
}

func (r *Runner) worker(host string) (result Result) {
	if isHostname(host) {
		ips, err := resolveDomain(host, r.Options.Resolvers)
		if err != nil {
			log.Warn("Failed to resolve domain", host)
			return
		}

		for _, ip := range ips {
			if isDNSEnabled(ip, time.Duration(r.Options.Timeout)*time.Second) {
				return Result{Host: ip + ":53"}
			}
		}
	} else {
		if isDNSEnabled(host, time.Duration(r.Options.Timeout)*time.Second) {
			return Result{Host: host + ":53"}
		}
	}

	return
}

func resolveDomain(domain string, resolvers []string) ([]string, error) {
	var ips []string
	dnsResolverProto := "udp"    // Protocol to use for the DNS resolver
	dnsResolverTimeoutMs := 5000 // Timeout (ms) for the DNS resolver (optional)
	var lastErr error            // Store the last error, if any

	// First, try resolving using the default resolver
	ipAddrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), domain)
	if err == nil {
		// DNS resolution succeeded using the default resolver, use the resolved IP
		for _, ipAddr := range ipAddrs {
			ips = append(ips, ipAddr.IP.String())
		}
		return ips, nil
	}

	lastErr = err // Store the error from the default resolver

	// Next, try resolving using the custom resolvers
	for _, dnsResolverIP := range resolvers {
		d := net.Dialer{
			Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
		}
		_, err := d.DialContext(context.Background(), dnsResolverProto, dnsResolverIP)
		if err == nil {
			// DNS resolution succeeded using the custom resolver, use the resolved IP
			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
				},
			}

			ipAddrs, err := resolver.LookupIPAddr(context.Background(), domain)
			if err != nil {
				// log.Warnf("Failed to resolve IP address for domain %s: %v", domain, err)
				return ips, err
			}

			for _, ipAddr := range ipAddrs {
				ips = append(ips, ipAddr.IP.String())
			}
			return ips, nil
		}
		lastErr = err // Store the error in case all resolvers fail
	}

	errorMessage := fmt.Sprintf("Failed to resolve IP address for domain: %s", domain)
	return ips, fmt.Errorf("%s: %w", errorMessage, lastErr)
}

func isDNSEnabled(ip string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(ip, "53"), timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func isHostname(hostname string) bool {
	return net.ParseIP(hostname) == nil
}

func (r *Runner) getDelay() time.Duration {
	if r.Options.DelayJitter != 0 {
		return time.Duration(r.Options.Delay + rand.Intn(r.Options.DelayJitter))
	}
	return time.Duration(r.Options.Delay)
}
