package puffin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const PidMax = 32768

// FuncExec is an Exec implementation that uses provided go functions
// rather than the os/exec package
type FuncExec struct {
	funcMap map[string]CmdFunc
	envs    map[string]string
}

// NewFuncExec creates a new FuncExec struct
func NewFuncExec(opts ...FuncExecOption) Exec {
	exec := &FuncExec{}

	for _, opt := range opts {
		opt(exec)
	}

	return exec
}

// Lookpath finds a function in the function map and returns its name.
// If the funcMap does not contain the named command (or a matching path),
// an error will be returned
//
// commands will be matched by their exact name first and then
// by a matching file path in random order
func (e *FuncExec) LookPath(file string) (string, error) {
	_, found := e.findFunc(file)
	if found != "" {
		return found, nil
	}

	if strings.Contains(file, "/") {
		return found, &exec.Error{Name: file, Err: &os.PathError{Op: "stat", Path: file, Err: syscall.ENOENT}}
	}

	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

// Command creates a new Cmd that uses execs funcMap rather than the shell for
// running commands
func (e *FuncExec) Command(name string, arg ...string) Cmd {
	cmd := &FuncCmd{
		path:  name,
		args:  append([]string{name}, arg...),
		fExec: e,
	}
	if filepath.Base(name) == name {
		lp, err := e.LookPath(name)
		if lp != "" {
			cmd.path = lp
		}
		if err != nil {
			cmd.err = err
		}
	}

	return cmd
}

// CommandContext works the same as Command except it includes a context that
// can be used to cancle the commands execution
func (e *FuncExec) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	cmd := e.Command(name, arg...)
	cmd.(*FuncCmd).ctx = ctx

	return cmd
}

// findFunc retrives a function and the command name from the func map
func (e *FuncExec) findFunc(name string) (CmdFunc, string) {
	// check if it's a simple member of the map
	if fn, ok := e.funcMap[name]; ok {
		return fn, name
	}

	// check if there's a path that matches
	// e.g. go -> /usr/local/go/bin/go
	for file, fn := range e.funcMap {
		if filepath.Base(file) == name {
			return fn, file
		}
	}

	// no match was found
	return nil, ""
}

// FuncExecOption can be used to configure the FuncExec struct
type FuncExecOption func(*FuncExec)

// WithFuncMap sets the func map used by all the commands created by this Exec
func WithFuncMap(funcs map[string]CmdFunc) FuncExecOption {
	return func(fExec *FuncExec) {
		fExec.funcMap = funcs
	}
}

// WithEnv sets the env used by all the commands created by this Exec
func WithEnv(envs map[string]string) FuncExecOption {
	return func(fExec *FuncExec) {
		fExec.envs = envs
	}
}

// CmdFunc is a function that will run based on the command name
type CmdFunc func(*FuncCmd) int

// FuncCmd is a Cmd that runs a CmdFnd rather than running in a shell
type FuncCmd struct {
	path string
	args []string
	env  map[string]string
	dir  string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	extraFiles   []*os.File
	sysProcAttr  *syscall.SysProcAttr
	process      *os.Process
	processState *os.ProcessState

	ctx context.Context
	err error

	ctxErr   chan error
	startErr error
	exitCode chan int
	fExec    *FuncExec
}

// CombinedOutput runs the command function and returns its combined standard
// output and standard error.
func (c *FuncCmd) CombinedOutput() ([]byte, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	b := lockableBuffer{ReadWriter: &bytes.Buffer{}}
	c.stdout = &b
	c.stderr = &b
	err := c.Run()
	return b.Bytes(), err
}

// Environ returns a copy of the environment in which the command function would be run
// as it is currently configured.
func (c *FuncCmd) Environ() []string {
	env := c.env
	if env == nil && c.fExec != nil {
		return fmtEnv(c.fExec.envs)
	}

	return fmtEnv(c.env)
}

// fmtEnv converts a map of env names and values into a slice of strings
func fmtEnv(env map[string]string) []string {
	var fmted []string
	for k, v := range env {
		fmted = append(fmted, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(fmted)

	return fmted
}

// Output runs the command function and returns its standard output.
func (c *FuncCmd) Output() ([]byte, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	stdout := lockableBuffer{ReadWriter: &bytes.Buffer{}}
	c.stdout = &stdout

	captureErr := c.stderr == nil
	if captureErr {
		// TODO: port prefixSuffixSaver so error messages are the same
		c.stderr = &lockableBuffer{ReadWriter: &bytes.Buffer{}}
	}

	err := c.Run()
	if err != nil && captureErr {
		if ee, ok := err.(*exec.ExitError); ok {
			ee.Stderr = c.stderr.(*lockableBuffer).Bytes()
		}
	}
	return stdout.Bytes(), err
}

// Run starts the specified command and waits for it to complete.
func (c *FuncCmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Start starts the specified command and returns imediately.
// Wait should be called to wait for the command to complete.
func (c *FuncCmd) Start() error {
	if c.path == "" && c.err == nil {
		c.err = errors.New("exec: no command")
	}
	if c.err != nil {
		return c.err
	}
	if c.Process() != nil {
		return errors.New("exec: already started")
	}

	// check if the context is already done
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			c.lock()
			return c.ctx.Err()
		default:
		}
	}

	c.process = &os.Process{Pid: rand.Intn(PidMax)}

	fn, _ := c.fExec.findFunc(c.path)
	if fn == nil {
		c.startErr = errors.New("exit status 1")
		return nil
	}

	done := make(chan struct{})
	// start the command function in a go routine
	go func() {
		c.exitCode <- fn(c)
		done <- struct{}{}
	}()

	// listen for the command to be canceled
	go func() {
		select {
		case <-c.ctx.Done():
			c.lock()
			c.ctxErr <- c.ctx.Err()
		case <-done:
			return
		}
	}()

	return nil
}

// StderrPipe returns a io.ReadCloser that is attached to the cmds Stderr
func (c *FuncCmd) StderrPipe() (io.ReadCloser, error) {
	if c.stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StderrPipe after process started")
	}

	buf := &lockableBuffer{ReadWriter: &bytes.Buffer{}}
	c.stderr = buf
	return buf, nil
}

// StdinPipe returns an io.WriteCloser that is attached to the cmds Stdin
func (c *FuncCmd) StdinPipe() (io.WriteCloser, error) {
	if c.stdin != nil {
		return nil, errors.New("exec: Stdin already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StdinPipe after process started")
	}

	buf := &lockableBuffer{ReadWriter: &bytes.Buffer{}}
	c.stderr = buf
	return buf, nil
}

// StdoutPipe returns an io.ReadCloser that is attached to  to the cmds Stdout
func (c *FuncCmd) StdoutPipe() (io.ReadCloser, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StdoutPipe after process started")
	}

	buf := &lockableBuffer{ReadWriter: &bytes.Buffer{}}
	c.stderr = buf
	return buf, nil
}

// String returns a human readable description of the cmd
func (c *FuncCmd) String() string {
	if c.err != nil {
		// failed to resolve path; report the original requested path (plus args)
		return strings.Join(c.args, " ")
	}
	// report the exact executable path (plus args)
	var b strings.Builder
	b.WriteString(c.path)
	for _, a := range c.args[1:] {
		b.WriteByte(' ')
		b.WriteString(a)
	}
	return b.String()
}

func (c *FuncCmd) Wait() error {
	if c.startErr != nil {
		return c.startErr
	}

	if c.process == nil {
		return errors.New("exec: not started")
	}
	if c.processState != nil {
		return errors.New("exec: Wait was already called")
	}
	c.processState = &os.ProcessState{}

	if c.ctxErr != nil {
		interruptErr := <-c.ctxErr
		if interruptErr != nil {
			return interruptErr
		}
	}

	exitCode := <-c.exitCode
	if exitCode == 0 {
		return nil
	}

	// TODO: what error do I return here?
	return nil
}

// lock, prevents further changes to the underlying commands buffers
func (c *FuncCmd) lock() {
	c.stdin.(*lockableBuffer).LockRead()
	c.stdout.(*lockableBuffer).LockWrite()
	c.stderr.(*lockableBuffer).LockWrite()
}

// Path returns the Cmd path
func (c *FuncCmd) Path() string {
	return c.path
}

// SetPath sets the Cmd path
func (c *FuncCmd) SetPath(path string) {
	c.path = path
}

// Args returns the Cmd args
func (c *FuncCmd) Args() []string {
	return c.args
}

// SetArgs sets the Cmd args
func (c *FuncCmd) SetArgs(args []string) {
	c.args = args
}

// Env returns the Cmd env
func (c *FuncCmd) Env() []string {
	return fmtEnv(c.env)
}

// SetEnv sets the Cmd env
func (c *FuncCmd) SetEnv(env []string) {
	for _, e := range env {
		nameVal := strings.Split(e, "=")
		name := nameVal[0]
		var val string
		if len(nameVal) == 1 {
			val = nameVal[1]
		}
		c.env[name] = val
	}
}

// Dir returns the Cmd working dir
func (c *FuncCmd) Dir() string {
	return c.dir
}

// SetDir sets the Cmd working dir
func (c *FuncCmd) SetDir(dir string) {
	c.dir = dir
}

// Stdin returns the Cmd Stdin
func (c *FuncCmd) Stdin() io.Reader {
	return c.stdin
}

// SetStdin sets the Cmd Stdin
func (c *FuncCmd) SetStdin(stdin io.Reader) {
	c.stdin = &lockableReader{Reader: stdin}
}

// Stdout returns the Cmd Stdout
func (c *FuncCmd) Stdout() io.Writer {
	if writer, ok := c.stdout.(*lockableWriter); ok {
		return writer.Writer
	}
	if buffer, ok := c.stdout.(*lockableBuffer); ok {
		return buffer.ReadWriter
	}

	return c.stdout
}

// SetStdout sets the Cmd Stdout
func (c *FuncCmd) SetStdout(stdout io.Writer) {
	c.stdout = &lockableWriter{Writer: stdout}
}

// Stderr returns the Cmd Stderr
func (c *FuncCmd) Stderr() io.Writer {
	if writer, ok := c.stderr.(*lockableWriter); ok {
		return writer.Writer
	}
	if buffer, ok := c.stderr.(*lockableBuffer); ok {
		return buffer.ReadWriter
	}

	return c.stderr
}

// SetStderr sets the Cmd Stderr
func (c *FuncCmd) SetStderr(stderr io.Writer) {
	c.stderr = &lockableWriter{Writer: stderr}
}

// ExtraFiles returns the Cmd extra files
func (c *FuncCmd) ExtraFiles() []*os.File {
	return c.extraFiles
}

// SetExtraFiles sets the Cmd extra files
func (c *FuncCmd) SetExtraFiles(extraFiles []*os.File) {
	c.extraFiles = extraFiles
}

// SysProcAttr returns the Cmd sys proc attr
func (c *FuncCmd) SysProcAttr() *syscall.SysProcAttr {
	return c.sysProcAttr
}

// Process returns the Cmd process
func (c *FuncCmd) Process() *os.Process {
	return c.process
}

// ProcessState returns the Cmd process state
func (c *FuncCmd) ProcessState() *os.ProcessState {
	return c.processState
}

// Err returns the Cmd err
func (c *FuncCmd) Err() error {
	return c.err
}
