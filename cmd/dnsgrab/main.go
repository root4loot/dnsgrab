package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/root4loot/dnsgrab"
	"github.com/root4loot/dnsgrab/pkg/log"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
)

const author = "@danielantonsen"
const version = "0.0.0"

type CLI struct {
	Host        string // target host
	Concurrency int    // number of concurrent requests
	Timeout     int    // Request timeout duration (in seconds)
	Delay       int    // delay between each request (in ms)
	DelayJitter int    // maximum jitter to add to delay (in ms)
	UserAgent   string // custom user-agent
	Infile      string // file containin targets (newline separated)
	Outfile     string // file to write results
	Resolvers   string // file containing resolvers (newline separated)
	Verbose     bool   // hide info
	Silence     bool   // suppress output from console
	Version     bool   // print version
	Help        bool   // print help
}

func main() {
	cli := newCLI()
	cli.parseFlags()
	cli.checkForExits()
	cli.setLogger()
	cli.run()
}

func (c *CLI) run() {
	runner := dnsgrab.NewRunner()
	runner.Options.Concurrency = c.Concurrency
	runner.Options.Timeout = c.Timeout
	runner.Options.Delay = c.Delay
	runner.Options.DelayJitter = c.DelayJitter
	runner.Options.Verbose = c.Verbose

	if c.hasResolversFile() {
		resolvers, _ := c.readFileLines(c.Resolvers)
		runner.Options.Resolvers = resolvers
	}

	var targets []string

	if c.hasStdin() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			runner.MultipleStream(scanner.Text())
			c.processResults(runner)
		}
	} else if c.hasInfile() {
		targets, _ = c.readFileLines(c.Infile)
	} else if c.hasTarget() {
		targets = c.getTargets()
	}

	if len(targets) <= c.Concurrency {
		results := dnsgrab.Multiple(targets)
		for _, result := range results {
			if result.Host != "" {
				if !c.Silence {
					fmt.Println(result.Host)
				}
				if c.hasOutfile() {
					c.writeToFile(result.Host)
				}
			}
		}
	} else {
		c.processResults(runner)
		runner.MultipleStream(targets...)
	}
}

func (c *CLI) processResults(dnsgrab *dnsgrab.Runner) {
	go func() {
		for result := range dnsgrab.Results {
			if result.Host != "" {
				if !c.Silence {
					fmt.Println(result.Host)
				}
				if c.hasOutfile() {
					c.writeToFile(result.Host)
				}
			}
		}
	}()
}

func newCLI() *CLI {
	return &CLI{}
}

func (c *CLI) banner() {
	fmt.Println("\ndnsgrab", version, "by", author)
	fmt.Println("")
}

func (c *CLI) usage() {
	w := tabwriter.NewWriter(os.Stdout, 2, 0, 3, ' ', 0)

	fmt.Fprintf(w, "Usage:\t%s [options] (-h <host> | -i <hosts.txt>)\n\n", os.Args[0])

	fmt.Fprintf(w, "\nTARGETTING:\n")
	fmt.Fprintf(w, "\t%s,   %s\t\t  %s\n", "-h", "--host", "target host (comma separated)")
	fmt.Fprintf(w, "\t%s,   %s\t\t  %s\n", "-i", "--infile", "file containing hosts (newline separated)")

	fmt.Fprintf(w, "\nCONFIGURATIONS:\n")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t\n", "-r", "--resolvers", "file containing list of resolvers")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d %s)\n", "-c", "--concurrency", "number of concurrent requests", dnsgrab.DefaultOptions().Concurrency, "requests")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d %s)\n", "-t", "--timeout", "max request timeout", dnsgrab.DefaultOptions().Timeout, "seconds")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d %s)\n", "-d", "--delay", "delay between requests", dnsgrab.DefaultOptions().Delay, "milliseconds")
	fmt.Fprintf(w, "\t%s, %s\t%s\t(Default: %d %s)\n", "-dj", "--delay-jitter", "max jitter between requests", dnsgrab.DefaultOptions().DelayJitter, "milliseconds")

	fmt.Fprintf(w, "\nOUTPUT:\n")
	fmt.Fprintf(w, "\t%s,  %s\t\t  %s\n", "-o", "--outfile", "output results to given file")
	fmt.Fprintf(w, "\t%s,  %s\t\t  %s\n", "-s", "--silence", "silence everything")
	fmt.Fprintf(w, "\t%s,  %s\t\t  %s\n", "-v", "--verbose", "verbose output")
	fmt.Fprintf(w, "\t%s   %s\t\t  %s\n", "  ", "--version", "display version")

	w.Flush()
	fmt.Println("")
}

func (c *CLI) parseFlags() {
	// TARGET
	flag.StringVar(&c.Host, "host", "", "")
	flag.StringVar(&c.Host, "h", "", "")
	flag.StringVar(&c.Infile, "i", "", "")
	flag.StringVar(&c.Infile, "infile", "", "")

	// CONFIGURATIONS
	flag.IntVar(&c.Concurrency, "concurrency", dnsgrab.DefaultOptions().Concurrency, "")
	flag.IntVar(&c.Concurrency, "c", dnsgrab.DefaultOptions().Concurrency, "")
	flag.IntVar(&c.Timeout, "timeout", dnsgrab.DefaultOptions().Timeout, "")
	flag.IntVar(&c.Timeout, "t", dnsgrab.DefaultOptions().Timeout, "")
	flag.IntVar(&c.Delay, "delay", dnsgrab.DefaultOptions().Delay, "")
	flag.IntVar(&c.Delay, "d", dnsgrab.DefaultOptions().Delay, "")
	flag.IntVar(&c.DelayJitter, "delay-jitter", dnsgrab.DefaultOptions().DelayJitter, "")
	flag.IntVar(&c.DelayJitter, "dj", dnsgrab.DefaultOptions().DelayJitter, "")
	flag.StringVar(&c.Resolvers, "resolvers", "", "")
	flag.StringVar(&c.Resolvers, "r", "", "")

	// OUTPUT
	flag.BoolVar(&c.Silence, "s", false, "")
	flag.BoolVar(&c.Silence, "silence", false, "")
	flag.StringVar(&c.Outfile, "o", "", "")
	flag.StringVar(&c.Outfile, "outfile", "", "")
	flag.BoolVar(&c.Verbose, "v", false, "")
	flag.BoolVar(&c.Verbose, "verbose", false, "")
	flag.BoolVar(&c.Help, "help", false, "")
	flag.BoolVar(&c.Version, "version", false, "")

	flag.Usage = func() {
		c.banner()
		c.usage()
	}
	flag.Parse()
}

func (c *CLI) setLogger() {
	if c.Silence {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelSilent)
	} else if c.Verbose {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	}
}

func (c *CLI) checkForExits() {
	if c.Help {
		c.banner()
		c.usage()
		os.Exit(0)
	}
	if c.Version {
		fmt.Println("dnsgrab ", version)
		os.Exit(0)
	}

	if !c.hasStdin() && !c.hasInfile() && !c.hasTarget() {
		fmt.Println("")
		log.Errorf("%s\n\n", "Missing host")
		c.usage()
	}
}

func (c *CLI) hasTarget() bool {
	return c.Host != ""
}

func (c *CLI) hasInfile() bool {
	return c.Infile != ""
}

func (c *CLI) hasResolversFile() bool {
	return c.Resolvers != ""
}

func (c *CLI) hasOutfile() bool {
	return c.Outfile != ""
}

func (c *CLI) hasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	mode := stat.Mode()

	isPipedFromChrDev := (mode & os.ModeCharDevice) == 0
	isPipedFromFIFO := (mode & os.ModeNamedPipe) != 0

	return isPipedFromChrDev || isPipedFromFIFO
}

func (c *CLI) readFileLines(path string) (lines []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return
}

func (c *CLI) writeToFile(lines ...string) {
	file, err := os.OpenFile(c.Outfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	for i := range lines {
		if _, err := file.WriteString(lines[i] + "\n"); err != nil {
			log.Errorf("could not write line to file: %v", err)
		}
	}
}

func (c *CLI) getTargets() (targets []string) {
	if c.hasTarget() {
		if strings.Contains(c.Host, ",") {
			c.Host = strings.ReplaceAll(c.Host, " ", "")
			targets = strings.Split(c.Host, ",")
		} else {
			targets = append(targets, c.Host)
		}
	}
	return
}
