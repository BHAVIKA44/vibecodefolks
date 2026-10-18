package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

func Handler(dir string, assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/changes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}
		changes, err := analysis.Analyze(dir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(changes)
	})
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux
}
