package main

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
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

func dumpRequest(c echo.Context) error {
	r := &DumpResult{}

	r.Path = c.Request().RequestURI
	r.Hostname, _ = os.Hostname()
	r.Time = time.Now().Format(time.RFC3339Nano)
	r.RemoteAddr = c.Request().RemoteAddr
	r.Headers = make(map[string]string)

	for name, values := range c.Request().Header {
		for _, v := range values {
			r.Headers[name] = v
		}
	}
	appName := os.Getenv("X_APP_NAME")
	if appName != "" {
		r.Headers["X-Application-Name"] = appName
	}

	return c.JSON(http.StatusOK, r)
}
