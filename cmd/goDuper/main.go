package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"goduper/internal/collect"
	"goduper/internal/model"
	"goduper/internal/opts"
	"goduper/internal/plex"
	"goduper/internal/report"
)

// Version is the CLI version. Override at build time with:
//
//	go build -ldflags "-X main.Version=v0.3.0" ./cmd/goDuper
var Version = "v0.5.2"

func main() {
	o := opts.Parse()

	if o.ShowVersion {
		fmt.Println("goDuper", Version)
		fmt.Println("github.com/srv1054/goDuper")
		return
	}

	ctx := context.Background()
	pc, err := plex.NewClient(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	out, err := collect.Run(ctx, pc, o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	// JSON to stdout
	enc := json.NewEncoder(os.Stdout)
	if o.Pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "FATAL:", err)
		os.Exit(1)
	}

	// Optional HTML report
	if o.HTMLOut != "" {
		if err := report.RenderHTML(out, o.Verify, o.HTMLOut); err != nil {
			fmt.Fprintln(os.Stderr, "WARN:", "write HTML:", err)
		} else if o.Verbose {
			fmt.Fprintln(os.Stderr, "HTML report written to", o.HTMLOut)
		}
	}

	_ = model.Output{} // keep import if optimizer gets cute
}
