// Package gstr1 prepares a portal-ready GSTR-1 upload from invoice data:
// importers for the official offline-tool CSV shapes and a simple invoice-level
// JSON, automatic Table 12 (HSN) and Table 13 (documents issued) computation,
// rate-wise B2CS consolidation, and the upload JSON emitter.
package gstr1

import "github.com/heypkv/djin/internal/gst"

// DefaultVersion is the GSTR-1 schema version stamped into the upload. The
// portal's own downloads carry this string.
const DefaultVersion = "GST3.2.1"

// Portal is the GSTR-1 upload document in the portal's JSON shape. Empty
// sections are omitted. Field order mirrors the portal's own output.
type Portal struct {
	Gstin    string     `json:"gstin"`
	Fp       string     `json:"fp"`
	Version  string     `json:"version"`
	Hash     string     `json:"hash"`
	B2b      []B2BCtin  `json:"b2b,omitempty"`
	B2cs     []B2CSRow  `json:"b2cs,omitempty"`
	B2cl     []B2CLPos  `json:"b2cl,omitempty"`
	Cdnr     []CDNRCtin `json:"cdnr,omitempty"`
	Cdnur    []CDNURRow `json:"cdnur,omitempty"`
	Exp      []EXPType  `json:"exp,omitempty"`
	Nil      *NilSec    `json:"nil,omitempty"`
	Hsn      *HSN       `json:"hsn,omitempty"`
	DocIssue *DocIssue  `json:"doc_issue,omitempty"`
}

// --- B2B (Table 4A) ---

type B2BCtin struct {
	Ctin string   `json:"ctin"`
	Inv  []B2BInv `json:"inv"`
}

type B2BInv struct {
	Inum   string     `json:"inum"`
	Idt    string     `json:"idt"`
	Val    gst.Amount `json:"val"`
	Pos    string     `json:"pos"`
	Rchrg  string     `json:"rchrg"`
	InvTyp string     `json:"inv_typ"`
	Itms   []Itm      `json:"itms"`
}

type Itm struct {
	Num    int    `json:"num"`
	ItmDet ItmDet `json:"itm_det"`
}

// ItmDet carries a rate-wise line of tax. For intra-state supply camt/samt are
// present and iamt is omitted; for inter-state the reverse. csamt (cess) is
// always present.
type ItmDet struct {
	Rt    float64     `json:"rt"`
	Txval gst.Amount  `json:"txval"`
	Iamt  *gst.Amount `json:"iamt,omitempty"`
	Camt  *gst.Amount `json:"camt,omitempty"`
	Samt  *gst.Amount `json:"samt,omitempty"`
	Csamt gst.Amount  `json:"csamt"`
}

// --- B2CS (Table 7) ---

type B2CSRow struct {
	SplyTy string      `json:"sply_ty"`
	Rt     float64     `json:"rt"`
	Typ    string      `json:"typ"`
	Pos    string      `json:"pos"`
	Txval  gst.Amount  `json:"txval"`
	Iamt   *gst.Amount `json:"iamt,omitempty"`
	Camt   *gst.Amount `json:"camt,omitempty"`
	Samt   *gst.Amount `json:"samt,omitempty"`
	Csamt  gst.Amount  `json:"csamt"`
}

// --- B2CL (Table 5) ---

type B2CLPos struct {
	Pos string    `json:"pos"`
	Inv []B2CLInv `json:"inv"`
}

type B2CLInv struct {
	Inum string     `json:"inum"`
	Idt  string     `json:"idt"`
	Val  gst.Amount `json:"val"`
	Itms []Itm      `json:"itms"`
}

// --- CDNR / CDNUR (Table 9B) ---

type CDNRCtin struct {
	Ctin string     `json:"ctin"`
	Nt   []CDNRNote `json:"nt"`
}

type CDNRNote struct {
	Ntty   string     `json:"ntty"`
	Nt_num string     `json:"nt_num"`
	Nt_dt  string     `json:"nt_dt"`
	Pos    string     `json:"pos"`
	Rchrg  string     `json:"rchrg"`
	InvTyp string     `json:"inv_typ"`
	Val    gst.Amount `json:"val"`
	Itms   []Itm      `json:"itms"`
}

type CDNURRow struct {
	Typ    string     `json:"typ"`
	Ntty   string     `json:"ntty"`
	Nt_num string     `json:"nt_num"`
	Nt_dt  string     `json:"nt_dt"`
	Pos    string     `json:"pos,omitempty"`
	Val    gst.Amount `json:"val"`
	Itms   []Itm      `json:"itms"`
}

// --- Exports (Table 6A) ---

type EXPType struct {
	ExpTyp string   `json:"exp_typ"`
	Inv    []EXPInv `json:"inv"`
}

type EXPInv struct {
	Inum    string     `json:"inum"`
	Idt     string     `json:"idt"`
	Val     gst.Amount `json:"val"`
	Sbpcode string     `json:"sbpcode,omitempty"`
	Sbnum   string     `json:"sbnum,omitempty"`
	Sbdt    string     `json:"sbdt,omitempty"`
	Itms    []ExpItm   `json:"itms"`
}

type ExpItm struct {
	Txval gst.Amount `json:"txval"`
	Rt    float64    `json:"rt"`
	Iamt  gst.Amount `json:"iamt"`
	Csamt gst.Amount `json:"csamt"`
}

// --- Nil/Exempt (Table 8) ---

type NilSec struct {
	InvType []NilRow `json:"inv"`
}

type NilRow struct {
	SplyTy   string     `json:"sply_ty"`
	NilAmt   gst.Amount `json:"nil_amt"`
	ExptAmt  gst.Amount `json:"expt_amt"`
	NgsupAmt gst.Amount `json:"ngsup_amt"`
}

// --- HSN (Table 12, post-May-2025 B2B/B2C split) ---

type HSN struct {
	HsnB2b []HSNRow `json:"hsn_b2b,omitempty"`
	HsnB2c []HSNRow `json:"hsn_b2c,omitempty"`
}

type HSNRow struct {
	Num   int        `json:"num"`
	HsnSc string     `json:"hsn_sc"`
	Desc  string     `json:"desc"`
	Uqc   string     `json:"uqc"`
	Qty   float64    `json:"qty"`
	Rt    float64    `json:"rt"`
	Txval gst.Amount `json:"txval"`
	Iamt  gst.Amount `json:"iamt"`
	Camt  gst.Amount `json:"camt"`
	Samt  gst.Amount `json:"samt"`
	Csamt gst.Amount `json:"csamt"`
}

// --- Documents issued (Table 13) ---

type DocIssue struct {
	DocDet []DocDet `json:"doc_det"`
}

type DocDet struct {
	DocNum int        `json:"doc_num"`
	Docs   []DocEntry `json:"docs"`
}

type DocEntry struct {
	Num      int    `json:"num"`
	From     string `json:"from"`
	To       string `json:"to"`
	Totnum   int    `json:"totnum"`
	Cancel   int    `json:"cancel"`
	NetIssue int    `json:"net_issue"`
}
