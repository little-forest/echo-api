package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type HttpHeader struct {
	Name  string
	Value string
}

func NewHttpHeader(name string, value string) *HttpHeader {
	h := &HttpHeader{
		Name:  name,
		Value: value}
	return h
}

type DumpResult struct {
	Path       string            `json:"path"`
	Time       string            `json:"time"`
	RemoteAddr string            `json:"remoteAddr"`
	Hostname   string            `json:"hostname"`
	Headers    map[string]string `json:"headers"`
}

func dumpRequest(w http.ResponseWriter, req *http.Request) {
	r := &DumpResult{}

	r.Path = req.RequestURI
	r.Hostname, _ = os.Hostname()
	r.Time = time.Now().Format(time.RFC3339Nano)
	r.RemoteAddr = req.RemoteAddr
	r.Headers = make(map[string]string)

	for name, values := range req.Header {
		for _, v := range values {
			r.Headers[name] = v
		}
	}
	appName := os.Getenv("X_APP_NAME")
	if appName != "" {
		r.Headers["X-Application-Name"] = appName
	}

	writeJSON(w, http.StatusOK, r)
}

// writeJSON writes v in the same form as echo's default JSON serializer
// (no charset, HTML escaped, trailing newline).
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
