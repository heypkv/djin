package gstr1

import (
	"fmt"
	"strings"
)

// Issue is a single validation failure that names the offending row and field
// — the opposite of the portal's opaque RET191xxx codes.
type Issue struct {
	Section string `json:"section"`         // e.g. "b2b", "input"
	Ref     string `json:"ref"`             // invoice / note number or row id
	Field   string `json:"field,omitempty"` // offending field
	Message string `json:"message"`
}

func (i Issue) String() string {
	loc := i.Section
	if i.Ref != "" {
		loc += " " + i.Ref
	}
	if i.Field != "" {
		loc += " [" + i.Field + "]"
	}
	return fmt.Sprintf("%s: %s", loc, i.Message)
}

// Report collects validation issues from import and computation.
type Report struct {
	Issues []Issue `json:"issues"`
}

// Add records a failure against a specific row and field.
func (r *Report) Add(section, ref, field, format string, args ...any) {
	r.Issues = append(r.Issues, Issue{
		Section: section,
		Ref:     ref,
		Field:   field,
		Message: fmt.Sprintf(format, args...),
	})
}

// OK reports whether the report is free of issues.
func (r *Report) OK() bool { return len(r.Issues) == 0 }

// Error renders every issue, one per line.
func (r *Report) Error() string {
	if r.OK() {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d validation issue(s):", len(r.Issues))
	for _, is := range r.Issues {
		b.WriteString("\n  - ")
		b.WriteString(is.String())
	}
	return b.String()
}
