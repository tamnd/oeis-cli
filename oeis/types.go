package oeis

import (
	"fmt"
	"strings"
)

// Sequence is the exported record emitted for every matching sequence.
type Sequence struct {
	ANumber  string   `json:"anumber"`  // "A000045"
	Name     string   `json:"name"`     // human-readable name
	Data     string   `json:"data"`     // comma-separated terms
	Keywords []string `json:"keywords"` // parsed from keyword field
	Author   string   `json:"author"`   // submitting author
	Offset   string   `json:"offset"`   // e.g. "0,3"
	XRefs    []string `json:"xrefs"`    // first 3 cross-references
	Comment  string   `json:"comment"`  // first comment, if any
	URL      string   `json:"url"`      // "https://oeis.org/A000045"
}

// wireSearchResp is the top-level JSON object returned by every API call.
type wireSearchResp struct {
	Greeting string         `json:"greeting"`
	Query    string         `json:"query"`
	Count    int            `json:"count"`
	Start    int            `json:"start"`
	Results  []wireSequence `json:"results"`
}

// wireSequence is one entry in the results array.
type wireSequence struct {
	Number    int      `json:"number"`
	ID        string   `json:"id"`
	Data      string   `json:"data"`
	Name      string   `json:"name"`
	Comment   []string `json:"comment"`
	Reference []string `json:"reference"`
	Link      []string `json:"link"`
	Formula   []string `json:"formula"`
	Example   []string `json:"example"`
	Keyword   string   `json:"keyword"`
	Offset    string   `json:"offset"`
	Author    string   `json:"author"`
	XRef      []string `json:"xref"`
}

// wireToSequence converts a wire sequence to the exported record.
func wireToSequence(w wireSequence) Sequence {
	anumber := fmt.Sprintf("A%06d", w.Number)

	var keywords []string
	if w.Keyword != "" {
		for _, k := range strings.Split(w.Keyword, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				keywords = append(keywords, k)
			}
		}
	}

	xrefs := w.XRef
	if len(xrefs) > 3 {
		xrefs = xrefs[:3]
	}

	var comment string
	if len(w.Comment) > 0 {
		comment = w.Comment[0]
	}

	return Sequence{
		ANumber:  anumber,
		Name:     w.Name,
		Data:     w.Data,
		Keywords: keywords,
		Author:   w.Author,
		Offset:   w.Offset,
		XRefs:    xrefs,
		Comment:  comment,
		URL:      "https://oeis.org/" + anumber,
	}
}
