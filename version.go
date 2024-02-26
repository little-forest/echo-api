package main

import "fmt"

// Will be fixed by LDFLAGS
var (
	version  = "dev"
	revision = "dev"
	date     = "unknown"
	osArch   = "unknown"
)

func showVersion() {
	fmt.Printf("echo-api version %s %s built from %s on %s\n", version, osArch, revision, date)
}
