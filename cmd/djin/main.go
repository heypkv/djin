// Command djin is the offline GST compliance CLI: prepare a portal-ready
// GSTR-1 upload JSON from invoice data, and serve a local web UI under the
// hey app contract. It follows the heypkv/kitsy ecosystem pattern — a single
// static binary with an embedded UI, distributed through hey.
package main

import (
	"fmt"
	"os"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "0.1.0"

const usageText = `djin — offline GST compliance tool

Usage:
  djin gstr1  build -i <input.json> [-o <upload.json>] [--summary]
  djin gstr1  import <section> -i <file.csv>   (b2b|b2cs|b2cl|cdnr|cdnur|exp|exempt|docs)
  djin ui     [--port <n>] [--no-open] [--json]
  djin version
  djin help

Flags (gstr1 build):
  -i, --in       invoice-level JSON input (@file or literal)
  -o, --out      write the portal upload JSON to a file (default: stdout)
      --summary  also print the pre-upload reconciliation summary to stderr

Examples:
  djin gstr1 build -i @invoices.json -o GSTR1_upload.json
  djin ui --port 0 --json
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "gstr1":
		err = cmdGSTR1(os.Args[2:])
	case "ui":
		err = cmdUI(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("djin " + version)
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "djin:", err)
		os.Exit(1)
	}
}
