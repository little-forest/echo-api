package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const defaultPort = 8080

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

	e := echo.New()

	// Setup middlewares
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			slog.Info("request", "remoteIP", v.RemoteIP, "method", v.Method, "status", v.Status, "uri", v.URI, "latency", v.Latency)
			return nil
		},
	}))
	e.Use(middleware.Secure())

	e.GET("/*", dumpRequest)

	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatal(err)
	}
}
