---
id: TASK-GSTR-1-CORE-IMPO-COM-1
short_id: 55693938654f
title: "GSTR-1 core: import, compute, emit"
type: feature
status: in_review
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: mvp
acceptance:
  - imports official-tool-compatible section CSVs (specs in
    C:/Users/pkvsi/gst_offline_tool/Section_wise_CSV_files/GSTR1) for B2B, B2CS,
    B2CL, CDNR/CDNUR, EXP, exempt, docs
  - auto-computes Table 12 HSN (B2B/B2C split, post-May-2025) and Table 13 from
    invoice rows; rate-wise B2CS consolidation
  - emits portal upload JSON (fp = last month for QRMP quarters;
    hsn_b2b/hsn_b2c; <=5MB enforced) matching the known-good
    GSTR1_10AAICH1439H1ZZ_062026.json as a golden test
  - validation report names invoice+field for every failure
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
