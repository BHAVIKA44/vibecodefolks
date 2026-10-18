package explain

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

// Quiz answers are AI interpretations, not verified claims about runtime behavior.
type Quiz struct {
	Questions []Question `json:"questions"`
}
type Question struct {
	ID           string      `json:"id"`
	Text         string      `json:"text"`
	Options      []string    `json:"options"`
	CorrectIndex *int        `json:"correct_index"`
	Explanation  string      `json:"explanation"`
	References   []Reference `json:"references"`
	Category     string      `json:"category"`
}

func quizOrNil(raw json.RawMessage, ev analysis.Evidence) *Quiz {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var q Quiz
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&q) != nil {
		return nil
	}
	var extra any
	if dec.Decode(&extra) != io.EOF || len(q.Questions) == 0 || len(q.Questions) > 5 {
		return nil
	}
	ids := map[string]bool{}
	for _, item := range q.Questions {
		if !validID.MatchString(item.ID) || ids[item.ID] || !bounded(item.Text, 500) || !bounded(item.Explanation, 1500) || len(item.Options) != 4 || item.CorrectIndex == nil || *item.CorrectIndex < 0 || *item.CorrectIndex > 3 || len(item.References) == 0 || len(item.References) > 10 {
			return nil
		}
		ids[item.ID] = true
		switch item.Category {
		case "change", "purpose", "relationship", "impact", "uncertainty":
		default:
			return nil
		}
		options := map[string]bool{}
		for _, option := range item.Options {
			normalized := strings.ToLower(strings.TrimSpace(option))
			if !bounded(option, 300) || options[normalized] {
				return nil
			}
			options[normalized] = true
		}
		for _, ref := range item.References {
			if !validRef(ref, ev) {
				return nil
			}
		}
		if item.Category == "relationship" {
			distinct := map[Reference]bool{}
			for _, ref := range item.References {
				distinct[ref] = true
			}
			if len(distinct) < 2 {
				return nil
			}
		}
	}
	return &q
}
