# djin feature set v1 — the offline compliance suite

djin is the compliance super-tool for Indian businesses: GST filing
preparation, DSC document signing, MCA filings, and the compliance calendar —
**offline, official-format, and pleasant**. It is *not* an accounting system;
it consumes what accounting/billing systems produce and emits what government
portals accept. The current government tooling (GST Returns Offline Tool,
emSigner) is a Java-era mess of installers, applets, and silent failures;
djin replaces that experience with one verified binary and a browser UI.

North star and distribution: see hey's
[north-star.md](https://github.com/heypkv/hey/blob/main/docs/north-star.md).
djin follows the ecosystem pattern exactly: Go single binary, embedded
Vite+React UI, hey app contract v0 (`djin ui` via `hey djin ui`), releases
via goreleaser consumed by hey's registry (entry already present).

## Feature 1 — GSTR-1 preparer (v1 slice, build first)

Prepare a portal-ready GSTR-1 upload JSON from invoice data, replacing the
official offline tool.

- **Import**: section-wise CSVs compatible with the official tool's
  templates (specs: `C:\Users\pkvsi\gst_offline_tool\Section_wise_CSV_files\GSTR1\`
  and the v2.2 Excel workbook), plus a simple invoice-level JSON/CSV format
  for people who never used the official tool.
- **Sections v1**: B2B (4A), B2CS (7), B2CL (5), CDNR/CDNUR (9B), exports
  (6A), nil/exempt (8), documents issued (13), HSN B2B/B2C split (12,
  post-May-2025 format). Amendments and ECO tables (9A/10/14/15) follow in
  v1.x.
- **Compute, don't ask**: Table 12 HSN summary and Table 13 document series
  derived automatically from the invoice lines; rate-wise B2CS
  consolidation; round-off checks; GSTIN checksum validation; POS/state-code
  tables; QRMP awareness (quarterly `fp` = last month of quarter — a real
  footgun we hit).
- **Output**: portal upload JSON (validated ≤ 5 MB, chunking guidance
  beyond), plus a human-readable summary that mirrors the portal's
  "Generate Summary" tiles for pre-upload reconciliation.
- **Validation report**: every rule failure points at the offending
  invoice/field — the exact opposite of the portal's RET191xxx runes.

## Feature 2 — DSC document signing (the emSigner killer)

- Sign PDFs (PAdES-style embedded signatures) and detached payloads with
  certificates from the **Windows certificate store** and **PKCS#11 USB
  tokens** (ePass, ProxKey, Watchdata — the usual DSC hardware).
- CLI (`djin sign pdf in.pdf --out signed.pdf`) and UI flows; signing audit
  log (what was signed, when, with which cert thumbprint).
- Later, exposed through the hey kernel so heypkv/kitsy *web* apps can
  request signatures locally — the capability browsers cannot have.

## Feature 3 — Invoice register & generator

Closes the loop we currently run by hand: maintain document series (e.g.
`PC27I-…`), generate GST-compliant invoice PDFs via the guten library,
record them in a local register (SQLite), and feed GSTR-1 Tables 4/7/12/13
directly from the register. One source of truth from invoice to filing.

## Feature 4 — Compliance calendar

Data-driven due-date rules (GSTR-1/3B incl. QRMP, TDS, MCA AOC-4/MGT-7,
advance tax) against a company profile; ICS export; reminders (later via
hey notifications / cloud).

## Architecture

```
djin/
  cmd/djin/            CLI: gstr1, sign, ui, version (guten-style dispatch)
  internal/gst/        types, rates, state codes, GSTIN validation, section rules
  internal/gstr1/      importers (csv/xlsx-compatible), computation, JSON emit
  internal/register/   invoice register (SQLite via modernc.org/sqlite, pure Go)
  internal/sign/       cert store + PKCS#11, PDF signing
  internal/webui/      embedded UI (built from ui/)
  ui/                  Vite + React source
  docs/
```

Storage is SQLite (pure-Go driver — keeps CGO off and goreleaser simple).
All formats importable/exportable; the user's data is theirs.

## v1 slice (first release)

Scaffold + `internal/gst` + GSTR-1 core for B2B/B2CS/HSN/docs + CSV import +
JSON export + summary + minimal UI (import → validate → review tiles →
download JSON) + contract v0. Ships as `v0.1.0`, distributed via
`hey djin ui` on day one.
