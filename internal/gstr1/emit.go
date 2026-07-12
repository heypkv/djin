package gstr1

import (
	"encoding/json"
	"fmt"
)

// MaxUploadBytes is the portal's per-file upload ceiling. Beyond it the return
// must be split; djin refuses to emit an over-size file rather than have the
// portal reject it opaquely.
const MaxUploadBytes = 5 << 20 // 5 MiB

// Marshal renders the upload as indented JSON and enforces the size ceiling.
func (p *Portal) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(b) > MaxUploadBytes {
		return nil, fmt.Errorf("upload is %d bytes, over the %d-byte portal limit — split the return by section or period", len(b), MaxUploadBytes)
	}
	return b, nil
}
