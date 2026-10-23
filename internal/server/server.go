package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
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
}
type state struct {
	mu       sync.Mutex
	requests map[string]prepared
	config   *explain.Config
}

func Handler(dir string, assets fs.FS, host string) http.Handler {
	s := &state{requests: map[string]prepared{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/provider", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !jsonRequest(w, r) {
			return
		}
		var in struct {
			Endpoint string `json:"endpoint"`
			Model    string `json:"model"`
			APIKey   string `json:"api_key"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&in) != nil {
			writeError(w, http.StatusBadRequest, "Invalid provider setup")
			return
		}
		cfg, err := explain.NewConfig(in.Endpoint, in.Model, in.APIKey)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.mu.Lock()
		s.config = &cfg
		s.mu.Unlock()
		writeJSON(w, map[string]bool{"configured": true})
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
		if ok && (p.config.Local || in.Consent) {
			delete(s.requests, in.Token)
		}
		s.mu.Unlock()
		if !ok || time.Now().After(p.expires) {
			writeError(w, http.StatusBadRequest, "Prepared evidence expired or was already used; try again")
			return
		}
		if !p.config.Local && !in.Consent {
			writeError(w, http.StatusForbidden, "Explicit consent is required before sending evidence to a remote provider")
			return
		}
		result, err := explain.NewProvider(p.config).Explain(r.Context(), p.evidence)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, map[string]any{"explanation": result, "evidence": p.evidence, "snapshot": p.snapshot, "captured_at": p.captured.UTC()})
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
		ev, err := analysis.Collect(dir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check the current evidence snapshot")
			return
		}
		data, err := json.Marshal(ev)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check the current evidence snapshot")
			return
		}
		digest := sha256.Sum256(data)
		writeJSON(w, map[string]bool{"fresh": hex.EncodeToString(digest[:]) == in.Snapshot})
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
