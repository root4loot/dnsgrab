package dnsgrab

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
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

// DefaultOptions returns default options
func DefaultOptions() *Options {
	publicresolvers, _ := publicresolvers.FetchResolversTrusted()

	return &Options{
		Concurrency: 10,
		Timeout:     3,
		Delay:       0,
		DelayJitter: 0,
		Resolvers:   publicresolvers,
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	options := DefaultOptions()
	return &Runner{
		Results: make(chan Result),
		Visited: make(map[string]bool),
		Options: *options,
	}
}

// Single runs dnsgrab against a single host
func Single(host string) (result Result) {
	// fmt.Println("Running single", host)
	r := NewRunner()
	r.Options.Concurrency = 1
	return r.worker(host)
}

// Multiple runs dnsgrab against multiple hosts
func Multiple(hosts []string) (results []Result) {
	// fmt.Println("Running multiple", hosts)
	r := NewRunner()
	if r.Options.Concurrency > len(hosts) {
		r.Options.Concurrency = len(hosts)
	}

	for _, host := range hosts {
		results = append(results, r.worker(host))
	}
	return
}

// MultipleStream runs dnsgrab against multiple hosts in async mode
func (r *Runner) MultipleStream(host ...string) {
	// fmt.Println("Running multiple stream", host)
	defer close(r.Results)

	if r.Options.Verbose {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	}

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, h := range host {
		if !r.Visited[h] {
			r.Visited[h] = true

			sem <- struct{}{}
			wg.Add(1)
			go func(u string) {
				defer func() { <-sem }()
				defer wg.Done()
				r.Results <- r.worker(u)
				time.Sleep(time.Millisecond * 100) // make room for processing results
			}(h)
			time.Sleep(r.getDelay() * time.Millisecond) // delay between requests
		}
	}
	wg.Wait()
}

func (r *Runner) worker(host string) (result Result) {
	// timeout := time.Duration(r.Options.Timeout) * time.Minute

	if isHostname(host) {
		ips, err := resolveDomain(host, r.Options.Resolvers)
		if err != nil {
			log.Printf("Failed to resolve domain %s: %v\n", host, err)
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
				// log.Printf("Failed to resolve IP address for domain %s: %v\n", domain, err)
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
