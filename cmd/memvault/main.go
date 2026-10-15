package main

import (
	"fmt"
	"memvault/internal/app"
	"os"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "memvault:", err)
		os.Exit(1)
	}
}
