package main

import "fmt"

// cmdGSTR1 is a thin dispatcher for the GSTR-1 preparer. Task 1 ships it as a
// stub; the real build/import subcommands land with internal/gstr1.
func cmdGSTR1(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: djin gstr1 <build|import> ... (see 'djin help')")
	}
	switch args[0] {
	case "build", "import":
		return fmt.Errorf("djin gstr1 %s: not implemented yet", args[0])
	default:
		return fmt.Errorf("unknown gstr1 subcommand %q (use build|import)", args[0])
	}
}
