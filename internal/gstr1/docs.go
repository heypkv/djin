package gstr1

import (
	"math/big"
	"sort"
)

// splitSeries splits a document number into its series prefix and the trailing
// numeric run: "KD27I-0000000001" -> ("KD27I-", "0000000001"). A number with
// no trailing digits yields the whole string as the prefix and an empty seq.
func splitSeries(num string) (prefix, seq string) {
	i := len(num)
	for i > 0 && num[i-1] >= '0' && num[i-1] <= '9' {
		i--
	}
	return num[:i], num[i:]
}

type docSeries struct {
	prefix  string
	fromNum string
	toNum   string
	fromSeq *big.Int
	toSeq   *big.Int
	total   int
	cancel  int
}

type docAccum struct {
	// keyed by (nature, prefix)
	order  []docSeriesKey
	series map[docSeriesKey]*docSeries
}

type docSeriesKey struct {
	nature int
	prefix string
}

func newDocAccum() *docAccum {
	return &docAccum{series: map[docSeriesKey]*docSeries{}}
}

func (a *docAccum) add(inv Invoice) {
	nature := inv.DocNature
	if nature == 0 {
		nature = 1 // invoices for outward supply
	}
	prefix, seqStr := splitSeries(inv.Number)
	seq := new(big.Int)
	if seqStr != "" {
		seq.SetString(seqStr, 10)
	}
	k := docSeriesKey{nature: nature, prefix: prefix}
	s := a.series[k]
	if s == nil {
		s = &docSeries{prefix: prefix, fromNum: inv.Number, toNum: inv.Number, fromSeq: new(big.Int).Set(seq), toSeq: new(big.Int).Set(seq)}
		a.series[k] = s
		a.order = append(a.order, k)
	}
	s.total++
	if inv.Cancelled {
		s.cancel++
	}
	if seq.Cmp(s.fromSeq) < 0 {
		s.fromSeq.Set(seq)
		s.fromNum = inv.Number
	}
	if seq.Cmp(s.toSeq) > 0 {
		s.toSeq.Set(seq)
		s.toNum = inv.Number
	}
}

// section renders Table 13, grouping series under their nature-of-document
// code. Within a nature, series are ordered by their "from" number.
func (a *docAccum) section() *DocIssue {
	if len(a.order) == 0 {
		return nil
	}
	byNature := map[int][]*docSeries{}
	var natures []int
	for _, k := range a.order {
		if _, seen := byNature[k.nature]; !seen {
			natures = append(natures, k.nature)
		}
		byNature[k.nature] = append(byNature[k.nature], a.series[k])
	}
	sort.Ints(natures)

	di := &DocIssue{}
	for _, nat := range natures {
		list := byNature[nat]
		sort.Slice(list, func(i, j int) bool { return list[i].fromNum < list[j].fromNum })
		det := DocDet{DocNum: nat}
		for i, s := range list {
			det.Docs = append(det.Docs, DocEntry{
				Num:      i + 1,
				From:     s.fromNum,
				To:       s.toNum,
				Totnum:   s.total,
				Cancel:   s.cancel,
				NetIssue: s.total - s.cancel,
			})
		}
		di.DocDet = append(di.DocDet, det)
	}
	return di
}
