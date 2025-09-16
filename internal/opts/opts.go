package opts

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Options struct {
	BaseURL      string
	Token        string
	SectionsCSV  string
	IncludeShows bool
	Deep         bool
	Pretty       bool
	Verify       bool
	InsecureTLS  bool
	Timeout      time.Duration
	Verbose      bool
	HTMLOut      string
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: goDuper -url http://HOST:32400 -token TOKEN [options]\n")
	flag.PrintDefaults()
}

func Parse() Options {
	var o Options
	flag.Usage = printUsage
	flag.StringVar(&o.BaseURL, "url", os.Getenv("PLEX_URL"), "Plex base URL (e.g. http://HOST:32400). Env: PLEX_URL")
	flag.StringVar(&o.Token, "token", os.Getenv("PLEX_TOKEN"), "Plex X-Plex-Token. Env: PLEX_TOKEN")
	flag.StringVar(&o.SectionsCSV, "sections", "", "Comma-separated section IDs to scan (skip auto-discovery if set)")
	flag.BoolVar(&o.IncludeShows, "include-shows", false, "Also scan show libraries (type=show)")
	flag.BoolVar(&o.Deep, "deep", true, "Deep fetch per item for complete Media/Part details (file paths, etc.)")
	flag.BoolVar(&o.Pretty, "pretty", true, "Pretty-print JSON output")
	flag.BoolVar(&o.Verify, "verify", true, "Verify on-disk files (adds checkFiles=1 to deep fetch, slower but accurate)")
	flag.BoolVar(&o.InsecureTLS, "insecure", false, "Skip TLS verification (self-signed HTTPS)")
	flag.DurationVar(&o.Timeout, "timeout", 20*time.Second, "HTTP timeout per request")
	flag.BoolVar(&o.Verbose, "v", false, "Verbose logs to stderr")
	flag.StringVar(&o.HTMLOut, "html-out", "", "Write a standalone HTML report to this file (in addition to JSON to stdout)")
	flag.Parse()

	if o.BaseURL == "" || o.Token == "" {
		fmt.Fprintln(os.Stderr, "ERROR: -url and -token are required (or set PLEX_URL/PLEX_TOKEN).")
		flag.Usage()
		os.Exit(2)
	}
	return o
}
