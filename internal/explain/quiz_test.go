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
		"missing questions":        func(q *Quiz) { q.Questions = nil },
		"duplicate ids":            func(q *Quiz) { q.Questions = append(q.Questions, q.Questions[0]) },
		"bad index":                func(q *Quiz) { v := 4; q.Questions[0].CorrectIndex = &v },
		"missing index":            func(q *Quiz) { q.Questions[0].CorrectIndex = nil },
		"duplicate options":        func(q *Quiz) { q.Questions[0].Options[1] = " LOGIN.TSX " },
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
