package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
	"github.com/BHAVIKA44/vibecodefolks/internal/explain"
)

type prepared struct {
	evidence analysis.Evidence
	config   explain.Config
	expires  time.Time
	snapshot string
	captured time.Time
	inUse    bool // true while a provider call is in progress; allows retry on failure
}
type quizSession struct {
	prepared prepared
	expires  time.Time
	busy     bool
	busyCtx  context.Context
	timer    *time.Timer
}
type state struct {
	mu       sync.Mutex
	requests map[string]prepared
	quizzes  map[string]quizSession
	config   *explain.Config
}

// Caller holds s.mu. Expired sessions release their captured evidence and provider
// configuration even when no later explanation request arrives.
func (s *state) addQuizSessionLocked(token string, p prepared, now time.Time, lifetime time.Duration) {
	session := quizSession{prepared: p, expires: now.Add(lifetime)}
	session.timer = time.AfterFunc(lifetime, func() {
		s.mu.Lock()
		delete(s.quizzes, token)
		s.mu.Unlock()
	})
	s.quizzes[token] = session
}

func Handler(dir string, assets fs.FS, host string) http.Handler {
	s := &state{requests: map[string]prepared{}, quizzes: map[string]quizSession{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/harnesses", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		type harnessStatus struct {
			Name      string `json:"name"`
			Installed bool   `json:"installed"`
			Supported bool   `json:"supported"`
		}
		candidates := []struct {
			name     string
			commands []string
		}{
			{"codex", []string{"codex"}}, {"claude", []string{"claude"}},
			{"gemini", []string{"gemini"}}, {"cursor", []string{"cursor-agent"}},
			{"antigravity", []string{"agy"}}, {"opencode", []string{"opencode"}},
			{"aider", []string{"aider"}}, {"goose", []string{"goose"}},
		}
		result := make([]harnessStatus, 0, len(candidates))
		for _, candidate := range candidates {
			installed := false
			for _, command := range candidate.commands {
				if _, err := exec.LookPath(command); err == nil {
					installed = true
					break
				}
			}
			result = append(result, harnessStatus{Name: candidate.name, Installed: installed, Supported: candidate.name == "codex"})
		}
		writeJSON(w, result)
	})
	mux.HandleFunc("/api/provider", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") || !jsonRequest(w, r) {
			return
		}
		var in struct {
			Endpoint string `json:"endpoint"`
			Model    string `json:"model"`
			APIKey   string `json:"api_key"`
			Harness  string `json:"harness"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&in) != nil {
			writeError(w, http.StatusBadRequest, "Invalid provider setup")
			return
		}
		cfg, err := explain.NewConfig(in.Endpoint, in.Model, in.APIKey)
		if in.Harness != "" {
			cfg, err = explain.NewHarnessConfig(in.Harness)
			model := strings.TrimSpace(in.Model)
			if err == nil && (len(model) > 200 || strings.HasPrefix(model, "-") || strings.ContainsAny(in.Model, "\r\n")) {
				err = errors.New("Invalid Codex model name")
			}
			cfg.Model = model
			if err == nil {
				if _, lookupErr := exec.LookPath("codex"); lookupErr != nil {
					writeError(w, http.StatusBadRequest, "Codex CLI is not installed or is missing from the server PATH")
					return
				}
			}
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.mu.Lock()
		s.config = &cfg
		s.mu.Unlock()
		writeJSON(w, map[string]bool{"configured": true})
	})
	mux.HandleFunc("/api/evidence", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		ev, err := analysis.Collect(dir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not collect local diff evidence")
			return
		}
		writeJSON(w, ev)
	})
	mux.HandleFunc("/api/changes", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		changes, err := analysis.Analyze(dir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, changes)
	})
	mux.HandleFunc("/api/explain/prepare", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") || !jsonRequest(w, r) {
			return
		}
		s.mu.Lock()
		configured := s.config
		s.mu.Unlock()
		cfg, err := explain.ConfigFromEnv()
		if configured != nil {
			cfg, err = *configured, nil
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ev, err := analysis.Collect(dir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Evidence collection failed: "+err.Error())
			return
		}
		if len(ev.Files) == 0 {
			writeError(w, http.StatusBadRequest, "No eligible text diff evidence available. Skipped files: "+skippedSummary(ev.Skipped))
			return
		}
		buf := make([]byte, 32)
		if _, err = rand.Read(buf); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not prepare request")
			return
		}
		token := hex.EncodeToString(buf)
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.requests {
			if now.After(v.expires) {
				delete(s.requests, k)
			}
		}
		if len(s.requests) >= 16 {
			s.mu.Unlock()
			writeError(w, http.StatusTooManyRequests, "Too many pending requests; try again later")
			return
		}
		snapshotData, _ := json.Marshal(ev)
		digest := sha256.Sum256(snapshotData)
		snapshot := hex.EncodeToString(digest[:])
		s.requests[token] = prepared{evidence: ev, config: cfg, expires: now.Add(5 * time.Minute), snapshot: snapshot, captured: now}
		s.mu.Unlock()
		files := make([]string, 0, len(ev.Files))
		for _, f := range ev.Files {
			files = append(files, f.Path)
		}
		writeJSON(w, map[string]any{"token": token, "endpoint": cfg.PublicEndpoint(), "model": cfg.Model, "local": cfg.Local, "scope": ev.Scope, "files": files, "evidence": ev, "skipped": ev.Skipped, "limitations": ev.Limitations, "expires_in_seconds": 300, "snapshot": snapshot, "captured_at": now.UTC()})
	})
	mux.HandleFunc("/api/explain", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") || !jsonRequest(w, r) {
			return
		}
		var in struct {
			Token   string `json:"token"`
			Consent bool   `json:"consent"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil || in.Token == "" {
			writeError(w, http.StatusBadRequest, "Invalid explanation request")
			return
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			writeError(w, http.StatusBadRequest, "Invalid explanation request")
			return
		}
		s.mu.Lock()
		p, ok := s.requests[in.Token]
		if !ok || time.Now().After(p.expires) {
			s.mu.Unlock()
			writeError(w, http.StatusBadRequest, "Prepared evidence expired or was already used; try again")
			return
		}
		if p.inUse {
			s.mu.Unlock()
			writeError(w, http.StatusBadRequest, "Prepared evidence expired or was already used; try again")
			return
		}
		if !p.config.Local && !in.Consent {
			s.mu.Unlock()
			writeError(w, http.StatusForbidden, "Explicit consent is required before sending evidence to a remote provider")
			return
		}
		// Mark the token as in-use to prevent concurrent reuse while still allowing
		// the user to retry on a transient provider failure (token is only deleted on success).
		p.inUse = true
		s.requests[in.Token] = p
		s.mu.Unlock()
		result, err := explain.ProviderFor(p.config).Explain(r.Context(), p.evidence)
		if err != nil {
			// Release the in-use lock so the user can retry with the same token.
			s.mu.Lock()
			if entry, still := s.requests[in.Token]; still && entry.inUse {
				entry.inUse = false
				s.requests[in.Token] = entry
			}
			s.mu.Unlock()
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		sessionBytes := make([]byte, 32)
		if _, err := rand.Read(sessionBytes); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not prepare quiz session")
			return
		}
		sessionToken := hex.EncodeToString(sessionBytes)
		s.mu.Lock()
		delete(s.requests, in.Token)
		now := time.Now()
		for key, session := range s.quizzes {
			if now.After(session.expires) && !session.busy {
				session.timer.Stop()
				delete(s.quizzes, key)
			}
		}
		if len(s.quizzes) >= 16 {
			var oldestKey string
			var oldest time.Time
			for key, session := range s.quizzes {
				if !session.busy && (oldestKey == "" || session.expires.Before(oldest)) {
					oldestKey, oldest = key, session.expires
				}
			}
			if oldestKey != "" {
				s.quizzes[oldestKey].timer.Stop()
				delete(s.quizzes, oldestKey)
			} else {
				sessionToken = ""
			}
		}
		if sessionToken != "" {
			s.addQuizSessionLocked(sessionToken, p, now, 30*time.Minute)
		}
		s.mu.Unlock()
		writeJSON(w, map[string]any{"explanation": result, "evidence": p.evidence, "snapshot": p.snapshot, "captured_at": p.captured.UTC(), "quiz_session": sessionToken, "quiz_local": p.config.Local, "quiz_provider": p.config.PublicEndpoint(), "quiz_model": p.config.Model})
	})
	mux.HandleFunc("/api/quiz/generate", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") || !jsonRequest(w, r) {
			return
		}
		var in struct {
			Session  string `json:"session"`
			Snapshot string `json:"snapshot"`
			Consent  bool   `json:"consent"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil || len(in.Session) != 64 || len(in.Snapshot) != 64 {
			writeError(w, http.StatusBadRequest, "Invalid quiz request")
			return
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			writeError(w, http.StatusBadRequest, "Invalid quiz request")
			return
		}
		s.mu.Lock()
		session, ok := s.quizzes[in.Session]
		if !ok || time.Now().After(session.expires) || session.prepared.snapshot != in.Snapshot {
			s.mu.Unlock()
			writeError(w, http.StatusBadRequest, "Quiz session expired or does not match this snapshot. Explain the changes again")
			return
		}
		if !session.prepared.config.Local && !in.Consent {
			s.mu.Unlock()
			writeError(w, http.StatusForbidden, "Explicit consent is required before sending evidence for quiz generation")
			return
		}
		for wait := 0; session.busy && session.busyCtx != nil && session.busyCtx.Err() != nil && wait < 100; wait++ {
			s.mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			s.mu.Lock()
			session, ok = s.quizzes[in.Session]
			if !ok {
				s.mu.Unlock()
				writeError(w, http.StatusBadRequest, "Quiz session expired or does not match this snapshot. Explain the changes again")
				return
			}
		}
		if session.busy {
			s.mu.Unlock()
			writeError(w, http.StatusConflict, "Quiz generation is already running")
			return
		}
		session.busy = true
		session.busyCtx = r.Context()
		s.quizzes[in.Session] = session
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			current, exists := s.quizzes[in.Session]
			if exists {
				current.busy = false
				current.busyCtx = nil
				s.quizzes[in.Session] = current
			}
			s.mu.Unlock()
		}()
		fresh, err := evidenceFresh(dir, in.Snapshot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check the current evidence snapshot")
			return
		}
		if !fresh {
			writeError(w, http.StatusConflict, "The evidence changed. Refresh and explain the changes again")
			return
		}
		quiz, diagnostic, err := explain.GenerateQuiz(r.Context(), session.prepared.config, session.prepared.evidence)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		fresh, err = evidenceFresh(dir, in.Snapshot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check the current evidence snapshot")
			return
		}
		if !fresh {
			writeError(w, http.StatusConflict, "The evidence changed during generation. Refresh and explain the changes again")
			return
		}
		writeJSON(w, map[string]any{"quiz": quiz, "quiz_diagnostic": diagnostic, "snapshot": in.Snapshot})
	})
	mux.HandleFunc("/api/quiz/fresh", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") || !jsonRequest(w, r) {
			return
		}
		var in struct {
			Snapshot string `json:"snapshot"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil || len(in.Snapshot) != 64 {
			writeError(w, http.StatusBadRequest, "Invalid snapshot")
			return
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			writeError(w, http.StatusBadRequest, "Invalid snapshot")
			return
		}
		fresh, err := evidenceFresh(dir, in.Snapshot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check the current evidence snapshot")
			return
		}
		writeJSON(w, map[string]bool{"fresh": fresh})
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Host != host {
			writeError(w, http.StatusForbidden, "Invalid host")
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+host {
			writeError(w, http.StatusForbidden, "Invalid origin")
			return
		}
		site := r.Header.Get("Sec-Fetch-Site")
		if site != "" && site != "same-origin" && site != "none" {
			writeError(w, http.StatusForbidden, "Cross-origin request denied")
			return
		}
		if r.Method == "POST" && origin != "http://"+host {
			writeError(w, http.StatusForbidden, "Same-origin request required")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func evidenceFresh(dir, snapshot string) (bool, error) {
	ev, err := analysis.Collect(dir)
	if err != nil {
		return false, err
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return false, err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]) == snapshot, nil
}

func skippedSummary(files []analysis.SkippedFile) string {
	if len(files) == 0 {
		return "none"
	}
	if len(files) > 20 {
		files = files[:20]
	}
	var parts []string
	for _, f := range files {
		parts = append(parts, f.Path+" ("+f.Reason+")")
	}
	return strings.Join(parts, ", ")
}
func method(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == allowed {
		return true
	}
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}
func jsonRequest(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "JSON required")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
