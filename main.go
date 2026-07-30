package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
)

/*
	Ver is the CLI version. Override at build time with:

go build -ldflags "-X main.Ver=v0.9.1" .
*/
var Ver = "v0.9.2"

func main() {
	o := Parse()

	// Show version and exit
	if o.ShowVersion {
		fmt.Println("goPlexr", Ver)
		fmt.Println("github.com/srv1054/goPlexr")
		return
	}

	// Basic validation
	pc, err := NewClient(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	// Collect duplicates
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	out, err := RunCollection(ctx, pc, o)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "CANCELED: scan interrupted")
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}
	for _, warning := range out.Warnings {
		fmt.Fprintf(
			os.Stderr,
			"WARN: %s for section %q (%s): %s\n",
			warning.Code,
			warning.SectionTitle,
			warning.SectionID,
			warning.Message,
		)
	}

	// JSON to stdout
	if o.JSONOut != "" {
		if err := writeJSONFile(o.JSONOut, out, o.Pretty); err != nil {
			fmt.Fprintln(os.Stderr, "WARN: write JSON:", err)
		} else if o.Verbose {
			fmt.Fprintln(os.Stderr, "JSON written to", o.JSONOut)
		}
	}

	// JSON to stdout (unless -quiet)
	if !o.Quiet {
		enc := json.NewEncoder(os.Stdout)
		if o.Pretty {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, "FATAL:", err)
			os.Exit(1)
		}
	}

	// Optional HTML report
	if o.HTMLOut != "" {
		if err := RenderHTML(out, o.Verify, o.IgnoreExtras, o.HTMLOut); err != nil {
			fmt.Fprintln(os.Stderr, "WARN:", "write HTML:", err)
		} else if o.Verbose {
			fmt.Fprintln(os.Stderr, "HTML report written to", o.HTMLOut)
		}
	}

	_ = Output{} // keep import if optimizer gets cute
}

// writeJSONFile writes the given value as JSON to the specified file path.
func writeJSONFile(path string, v any, pretty bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}
