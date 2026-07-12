package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/heypkv/djin/internal/gst"
	"github.com/heypkv/djin/internal/gstr1"
)

// cmdGSTR1 dispatches the GSTR-1 preparer subcommands.
func cmdGSTR1(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: djin gstr1 <build|import> ... (see 'djin help')")
	}
	switch args[0] {
	case "build":
		return gstr1Build(args[1:])
	case "import":
		return gstr1Import(args[1:])
	default:
		return fmt.Errorf("unknown gstr1 subcommand %q (use build|import)", args[0])
	}
}

// loadArg returns s, or the contents of the file when s begins with '@',
// stripping a leading UTF-8 BOM (Windows editors add one).
func loadArg(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		b, err := os.ReadFile(s[1:])
		if err != nil {
			return "", err
		}
		return strings.TrimPrefix(string(b), "\ufeff"), nil
	}
	return s, nil
}

// gstr1Build reads invoice-level JSON, computes the upload, and writes it.
func gstr1Build(args []string) error {
	var in, out string
	var summary bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--in":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for %s", args[i-1])
			}
			in = args[i]
		case "-o", "--out":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for %s", args[i-1])
			}
			out = args[i]
		case "--summary":
			summary = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if in == "" {
		return fmt.Errorf("gstr1 build requires -i <input.json>")
	}
	raw, err := loadArg(in)
	if err != nil {
		return err
	}
	var input gstr1.ReturnInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return fmt.Errorf("parse input: %w", err)
	}
	portal, rep := gstr1.Build(input)
	if !rep.OK() {
		return fmt.Errorf("validation failed:\n%s", rep.Error())
	}
	b, err := portal.Marshal()
	if err != nil {
		return err
	}
	if out == "" {
		fmt.Println(string(b))
	} else {
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", out, len(b))
	}
	if summary {
		fmt.Fprintln(os.Stderr, gstr1.Summarize(portal).String())
	}
	return nil
}

// gstr1Import parses one official section CSV and prints its portal JSON.
func gstr1Import(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: djin gstr1 import <section> -i <file.csv> [--gstin <gstin> | --state <code>]")
	}
	section := args[0]
	var in, gstin, state string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-i", "--in":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for -i")
			}
			in = args[i]
		case "--gstin":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for --gstin")
			}
			gstin = args[i]
		case "--state":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for --state")
			}
			state = args[i]
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if in == "" {
		return fmt.Errorf("gstr1 import requires -i <file.csv>")
	}
	if state == "" && gstin != "" {
		state = gst.StateCodeOf(gstin)
	}
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	defer f.Close()

	var sectionData any
	var rep *gstr1.Report
	switch strings.ToLower(section) {
	case "b2b":
		sectionData, rep, err = gstr1.ImportB2B(f, state)
	case "b2cs":
		sectionData, rep, err = gstr1.ImportB2CS(f, state)
	case "b2cl":
		sectionData, rep, err = gstr1.ImportB2CL(f)
	case "cdnr":
		sectionData, rep, err = gstr1.ImportCDNR(f, state)
	case "cdnur":
		sectionData, rep, err = gstr1.ImportCDNUR(f)
	case "exp":
		sectionData, rep, err = gstr1.ImportEXP(f)
	case "exempt", "exemp", "nil":
		sectionData, rep, err = gstr1.ImportExempt(f)
	case "docs":
		sectionData, rep, err = gstr1.ImportDocs(f)
	default:
		return fmt.Errorf("unknown section %q (b2b|b2cs|b2cl|cdnr|cdnur|exp|exempt|docs)", section)
	}
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(sectionData, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	if rep != nil && !rep.OK() {
		fmt.Fprintln(os.Stderr, rep.Error())
	}
	return nil
}
