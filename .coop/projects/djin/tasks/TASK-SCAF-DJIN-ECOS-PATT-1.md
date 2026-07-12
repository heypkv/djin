---
id: TASK-SCAF-DJIN-ECOS-PATT-1
short_id: 979c94e463de
title: Scaffold djin on the ecosystem pattern
type: feature
status: in_review
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: mvp
acceptance:
  - Go module github.com/heypkv/djin; cmd/djin dispatch (guten-style); ui/
    Vite+React embedded via internal/webui; djin ui implements hey app contract
    v0 (--port 0 --json handshake, /healthz, /hey/shutdown, originGuard)
  - goreleaser + CI mirroring hey conventions; hey djin ui works from a local
    build via seeded cache
  - go vet + go test green on the skeleton
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
