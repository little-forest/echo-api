package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const defaultPort = 8080

const allowHeaderValue = "OPTIONS, GET"

var portArg int
var showVersionArg bool

func init() {
	flag.IntVar(&portArg, "p", defaultPort, "listening port number")
	flag.IntVar(&portArg, "port", defaultPort, "listening port number")
	flag.BoolVar(&showVersionArg, "v", false, "show version")
	flag.BoolVar(&showVersionArg, "version", false, "show version")
}

func getPortNumber() int {
	var port int
	portStr := os.Getenv("PORT")
	if portStr == "" {
		port = portArg
	} else {
		port, _ = strconv.Atoi(portStr)
	}
	return port
}

func main() {
	flag.Parse()
	if showVersionArg {
		showVersion()
		os.Exit(0)
	}

	port := getPortNumber()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler:     newHandler(),
		ReadTimeout: 30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("http server started", "address", ln.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
	}
}

// newHandler returns the whole application including middlewares.
// It is also used from tests.
func newHandler() http.Handler {
	return requestLogger(secureHeaders(http.HandlerFunc(route)))
}

// route dispatches every path by method only. http.ServeMux is not used
// because it redirects unclean paths and answers HEAD for GET routes.
func route(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		dumpRequest(w, r)
	case http.MethodOptions:
		w.Header().Set("Allow", allowHeaderValue)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodHead:
		w.Header().Set("Allow", allowHeaderValue)
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		w.Header().Set("Allow", allowHeaderValue)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method Not Allowed"})
	}
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteIP = r.RemoteAddr
		}
		slog.Info("request", "remoteIP", remoteIP, "method", r.Method, "status", status, "uri", r.RequestURI, "latency", time.Since(start))
	})
}
