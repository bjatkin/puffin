package puffin

import "context"

// Exec is the interface that matches the go exec package https://pkg.go.dev/os/exec
type Exec interface {
	// LookPath searches for an executable named file, if found
	// the path to the file is returned, if not an error is returned
	LookPath(file string) (string, error)

	// Command returns the Cmd struct to execute the named program
	// or function with the given arguments
	Command(name string, arg ...string) Cmd

	// CommandContext is like Command but includes a context
	// which can be used to kill the process
	CommandContext(ctx context.Context, name string, arg ...string) Cmd
}
