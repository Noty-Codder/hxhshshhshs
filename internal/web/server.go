// Package web serves the embedded banner image and a health endpoint.
// Telegram fetches rich-message images by URL, and Heroku/Railway/Render
// need a process that binds $PORT, so one tiny server solves both.
package web

import (
	_ "embed"
	"log/slog"
	"net/http"
)

//go:embed banner.jpg
var banner []byte

// Start launches the HTTP server in the background.
func Start(port string) {
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/banner.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(banner)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("IstkharMusic is running"))
	})
	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			slog.Warn("[web] server stopped", "error", err)
		}
	}()
}
