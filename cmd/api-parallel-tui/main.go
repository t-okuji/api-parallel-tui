package main

import (
	"fmt"
	"os"

	"api-parallel-tui/internal/app"
)

func main() {
	p := app.NewProgram()
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "program error: %v\n", err)
		os.Exit(1)
	}
}
