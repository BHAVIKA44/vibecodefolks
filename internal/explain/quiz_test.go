package explain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

func testQuiz() Quiz {
	n := 1
	return Quiz{Questions: []Question{{ID: "q1", Text: "Which file was added?", Options: []string{"login.tsx", "api.py", "old.java", "none"}, CorrectIndex: &n, Explanation: "Git evidence includes the added API file.", Category: "change", References: []Reference{{Path: "api.py", Side: "new", Start: 8, End: 8}}}}}
}
func TestQuizValidation(t *testing.T) {
	ev := graphFixture()
	cases := map[string]func(*Quiz){
		"missing questions": func(q *Quiz) { q.Questions = nil },
		"duplicate ids":     func(q *Quiz) { q.Questions = append(q.Questions, q.Questions[0]) },
		"duplicate question wording": func(q *Quiz) {
			duplicate := q.Questions[0]
			duplicate.ID = "q2"
			duplicate.Text = "  WHICH  file was added? "
			q.Questions = append(q.Questions, duplicate)
		},
		"bad index":                func(q *Quiz) { v := 4; q.Questions[0].CorrectIndex = &v },
		"missing index":            func(q *Quiz) { q.Questions[0].CorrectIndex = nil },
		"duplicate options":        func(q *Quiz) { q.Questions[0].Options[1] = " login.tsx " },
		"few options":              func(q *Quiz) { q.Questions[0].Options = q.Questions[0].Options[:3] },
		"fabricated file":          func(q *Quiz) { q.Questions[0].References[0].Path = "fake.go" },
		"wrong side":               func(q *Quiz) { q.Questions[0].References[0].Side = "old" },
		"wrong line":               func(q *Quiz) { q.Questions[0].References[0].Start = 90; q.Questions[0].References[0].End = 90 },
		"category":                 func(q *Quiz) { q.Questions[0].Category = "trivia" },
		"unsupported relationship": func(q *Quiz) { q.Questions[0].Category = "relationship" },
		"excessive": func(q *Quiz) {
			q.Questions = append(q.Questions, q.Questions[0], q.Questions[0], q.Questions[0], q.Questions[0], q.Questions[0])
		},
		"empty option": func(q *Quiz) { q.Questions[0].Options[0] = " " },
		"long text":    func(q *Quiz) { q.Questions[0].Text = strings.Repeat("a", 501) },
	}
	raw, _ := json.Marshal(testQuiz())
	if quizOrNil(raw, ev) == nil {
		t.Fatal("valid quiz rejected")
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			q := testQuiz()
			mutate(&q)
			raw, _ := json.Marshal(q)
			if quizOrNil(raw, ev) != nil {
				t.Fatal("invalid quiz accepted")
			}
		})
	}
	if quizOrNil(nil, ev) != nil || quizOrNil([]byte(`{"questions":[]}`), ev) != nil || quizOrNil(raw, graphFixtureEmpty()) != nil {
		t.Fatal("insufficient evidence accepted")
	}
}
func graphFixtureEmpty() (ev analysis.Evidence) { return }
func TestOptionalQuizPreservesExplanation(t *testing.T) {
	for _, suffix := range []string{"", `,"quiz":{"questions":"bad"}`, `,"quiz":{"questions":[]}`} {
		content := strings.TrimSuffix(answer("main.go"), "}") + suffix + "}"
		server := providerServer(t, 200, content)
		cfg, _ := NewConfig(server.URL, "test", "")
		got, err := NewProvider(cfg).Explain(context.Background(), fixture())
		server.Close()
		if err != nil || got.Summary != "A change" || got.Graph == nil || got.Quiz != nil {
			t.Fatalf("optional quiz: %+v %v", got, err)
		}
	}
}

func TestProviderReturnsQuizWithSameRequest(t *testing.T) {
	q := testQuiz()
	q.Questions[0].References = []Reference{{Path: "main.go", Side: "new", Start: 2, End: 2}}
	raw, _ := json.Marshal(q)
	content := strings.TrimSuffix(answer("main.go"), "}") + `,"quiz":` + string(raw) + "}"
	server := providerServer(t, 200, content)
	defer server.Close()
	cfg, _ := NewConfig(server.URL, "test", "")
	got, err := NewProvider(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Quiz == nil || len(got.Quiz.Questions) != 1 {
		t.Fatalf("valid quiz: %+v %v", got.Quiz, err)
	}
}

func TestQuizDiagnostics(t *testing.T) {
	ev := graphFixture()
	valid := testQuiz()
	raw, _ := json.Marshal(valid)
	cases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"missing", nil, "missing"},
		{"null", json.RawMessage("null"), "model_null"},
		{"empty", json.RawMessage(`{"questions":[]}`), "empty"},
		{"null questions", json.RawMessage(`{"questions":null}`), "invalid_structure"},
		{"missing questions", json.RawMessage(`{}`), "invalid_structure"},
		{"malformed", json.RawMessage(`{"questions":"wrong"}`), "invalid_structure"},
		{"valid", raw, "available"},
	}
	invalidReference := testQuiz()
	invalidReference.Questions[0].References[0].Path = "absent"
	bad, _ := json.Marshal(invalidReference)
	cases = append(cases, struct {
		name string
		raw  json.RawMessage
		want string
	}{"reference", bad, "invalid_reference"})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quiz, status := validateQuiz(tc.raw, ev)
			if status != tc.want || (quiz != nil) != (tc.want == "available") {
				t.Fatalf("quiz=%v status=%q, want %q", quiz, status, tc.want)
			}
		})
	}
}

func TestQuizDiagnosticPreservesExplanation(t *testing.T) {
	for _, tc := range []struct{ suffix, want string }{
		{"", "missing"},
		{`,"quiz":null`, "model_null"},
		{`,"quiz":{"questions":[]}`, "empty"},
		{`,"quiz":{"questions":"bad"}`, "invalid_structure"},
	} {
		content := strings.TrimSuffix(answer("main.go"), "}") + tc.suffix + "}"
		result, err := decodeExplanation(content, fixture())
		if err != nil || result.Summary != "A change" || result.Quiz != nil || result.QuizDiagnostic != tc.want {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	}
}

func TestDedicatedQuizValidation(t *testing.T) {
	q := testQuiz()
	q.Questions[0].OptionExplanations = []string{"First is wrong", "Second follows", "Third is wrong", "Fourth is wrong"}
	raw, _ := json.Marshal(q)
	if quiz, diagnostic, err := decodeDedicatedQuiz(string(raw), graphFixture()); err != nil || diagnostic != "available" || quiz == nil {
		t.Fatalf("valid response: quiz=%v diagnostic=%q err=%v", quiz, diagnostic, err)
	}
	q.Questions[0].OptionExplanations = []string{"only one"}
	raw, _ = json.Marshal(q)
	if quiz, diagnostic, err := decodeDedicatedQuiz(string(raw), graphFixture()); err != nil || diagnostic != "invalid_structure" || quiz != nil {
		t.Fatalf("invalid feedback: quiz=%v diagnostic=%q err=%v", quiz, diagnostic, err)
	}
	q.Questions[0].OptionExplanations = []string{"first", "second", "third", "fourth"}
	q.Questions[0].References[0].Path = "missing.go"
	raw, _ = json.Marshal(q)
	if quiz, diagnostic, err := decodeDedicatedQuiz(string(raw), graphFixture()); err != nil || diagnostic != "invalid_reference" || quiz != nil {
		t.Fatalf("invalid citation: quiz=%v diagnostic=%q err=%v", quiz, diagnostic, err)
	}
	for _, raw := range []string{`{"questions":[]}`, `{"questions":null}`, `not-json`} {
		if quiz, diagnostic, err := decodeDedicatedQuiz(raw, graphFixture()); err != nil || diagnostic == "available" || quiz != nil {
			t.Fatalf("unavailable response: quiz=%v diagnostic=%q err=%v", quiz, diagnostic, err)
		}
	}
}

func TestDedicatedQuizDistinctFeedbackAndCaseSensitiveOutcomes(t *testing.T) {
	q := testQuiz()
	q.Questions[0].Options = []string{"returns \"OK\"", "returns \"ok\"", "returns an error", "returns nothing"}
	q.Questions[0].OptionExplanations = []string{"Upper case result", "Lower case result", "Error path", "No result"}
	raw, _ := json.Marshal(q)
	if quiz, diagnostic, _ := decodeDedicatedQuiz(string(raw), graphFixture()); quiz == nil || diagnostic != "available" {
		t.Fatalf("case-sensitive outcomes rejected: %s", diagnostic)
	}
	q.Questions[0].OptionExplanations[2] = " Lower case result "
	raw, _ = json.Marshal(q)
	if quiz, diagnostic, _ := decodeDedicatedQuiz(string(raw), graphFixture()); quiz != nil || diagnostic != "invalid_structure" {
		t.Fatalf("duplicate feedback accepted: %s", diagnostic)
	}
	q.Questions[0].OptionExplanations = []string{"Correct: option 0", "Incorrect: option 1", "Incorrect: option 2", "Incorrect: option 3"}
	raw, _ = json.Marshal(q)
	if quiz, diagnostic, _ := decodeDedicatedQuiz(string(raw), graphFixture()); quiz != nil || diagnostic != "invalid_structure" {
		t.Fatalf("contradictory correct_index vs option_explanations accepted: %s", diagnostic)
	}
}
