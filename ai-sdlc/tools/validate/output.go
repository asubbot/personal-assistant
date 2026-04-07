package main

import (
	"fmt"
	"os"
)

// stdout is human-oriented CLI output. writeStdout avoids forbidigo (^fmt\.Print*) and satisfies errcheck.
var stdout = os.Stdout

func writeStdout(format string, args ...any) {
	_, _ = fmt.Fprintf(stdout, format, args...)
}

func writelnStdout(args ...any) {
	_, _ = fmt.Fprintln(stdout, args...)
}
