package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	e.Use(middleware.Logger())
	e.Use(middleware.Secure())

	e.GET("/*", dumpRequest)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
