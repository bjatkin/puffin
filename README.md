# Puffin 
Puffin abstracts the [os/exec go package](https://pkg.go.dev/os/exec).
It can be used replace calls to the underlying shell with calls to go functions.
This can be great for testing or for use cases where a simulated shell would be prefered over a real one.

# Quick Start
Puffin is designed to be easy to incoporate into your existing codebase.
Lets take the following code as an example.

```go
package main

import (
    "log"
    "os/exec"
)

func main() {
    clean, err := branchIsClean()
    if err != nil {
        log.Fatalln(err)
    }

    log.Printf("clean: %v\n", clean)
}

func branchIsClean() (bool error) {
    cmd := exec.Command("git", "status", "--porcelain")
    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}
```

This code and be re-written to use Puffin by doing the following.
```go
package main

import (
    "log"

	"github.com/bjatkin/puffin"
)

func main() {
    clean, err := branchIsClean(puffin.NewOsExec())
    if err != nil {
        log.Fatalln(err)
    }

    log.Printf("clean: %v\n", clean)
}

func branchIsClean(exec puffin.Exec) (bool error) {
    cmd := exec.Command("git", "status", "--porcelain")
    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}
```

A few things to note about this new code.
1) We've shadowed the exec package name with a `puffin.Exec` argument.
    This ensures that any code that was using the `os/exec` package previously will now use puffin instead.
2) Previously, thie code was using a global dependency `os/exec`.
    Now, however, the `puffin.Exec` dependency is being indected into our function.
3) `puffin.Exec` is an interface.
    `puffin.NewOsExec` is one implementation of that interface (there are others as well).
    This specific implementation behaves in the same way as the `os/exec` package.
    This means that while our code has changed, it's behavior has not.

The nice thing about this refactor is now we can test the `branchIsClean` function much more easily.
Take the following code as an example.

```go
package main

import (
	"testing"

	"github.com/bjatkin/puffin"
)

func Test_branchIsClean(t *testing.T) {
	tests := []struct {
		name    string
		cmdFunc puffin.CmdFunc
		want    bool
	}{
		{
			"is clean",
			func(fc *puffin.FuncCmd) int {
				return 0
			},
			true,
		},
		{
			"is dirty",
			func(fc *puffin.FuncCmd) int {
				fc.Stdout().Write([]byte("M README.md"))
				return 0
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := puffin.NewFuncExec(
				puffin.WithFuncMap(map[string]puffin.CmdFunc{
					"git": tt.cmdFunc,
				}),
			)
			got, err := branchIsClean(exec)
			if err != nil {
				t.Fatalf("branchIsClean() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("branchIsClean() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

Notice that this code uses `puffin.NewFuncExec` as it's implementation of the `puffin.Exec` interface.
This interfaces is a "simulated" shell and uses a `funcMap` that maps command names to go functions.
This allows exec commands, like `git`, to have consistent, easily configurable behaviors which is great for writing tests around code that uses exec commands.
Just remember, the point of these tests is to test the behavior of your code surrounding your tests, not the behavior the the commands themselves.