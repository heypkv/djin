# djin — agent router

**Status: PARKED (2026-07-13).** Active feature development is paused while the
ecosystem focus is on `hey` distribution. This is a pause, not abandonment —
resume from the coop `mvp` track when unparked.

## What djin is

Offline GST/compliance super-tool for Indian businesses (NOT an accounting
system). Go single binary + embedded web UI, distributed via `hey djin`.
Follows the ecosystem pattern (see
[hey's north star](https://github.com/kitsyai/hey/blob/main/docs/north-star.md)).
Full spec: [docs/feature-set-v1.md](docs/feature-set-v1.md).

## Shipped (v0.1.0)

- `internal/gst` — paise-integer money, GSTIN mod-36 checksum, rate/state tables.
- `internal/gstr1` — CSV importers (official offline-tool shapes), auto Table 12
  HSN (B2B/B2C split) + Table 13, portal JSON emit. **Golden test reproduces a
  real accepted filing** (`GSTR1_10AAICH1439H1ZZ_062026.json`).
- `djin gstr1 build|import` CLI; `djin ui` implements the hey app contract v0
  (placeholder UI only).

## Pending (coop `mvp` track — parked)

- `TASK-GSTR-1-UI-IMPO-VALI-1` — real Vite+React GSTR-1 UI (import → validate →
  summary tiles → download). Mirror guten's `cli/ui` + `cli/cmd/guten/ui.go`.
- `TASK-DSC-SIGN-SPIK-CERT-1` — DSC signing spike (Windows cert store + PKCS#11
  tokens). The emSigner killer.
- `TASK-INVO-REGI-GUTE-POWE-1` — invoice register (SQLite, pure-Go
  modernc.org/sqlite) + guten-powered PDF generation.

## To resume

`coop list tasks` in this repo, `coop show <id>`, then start from the GSTR-1 UI.
```
