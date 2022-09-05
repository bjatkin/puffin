package puffin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestExec_LookPath(t *testing.T) {
	type fields struct {
		funcMap map[string]CmdFunc
	}
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			"missing func",
			fields{
				funcMap: map[string]CmdFunc{},
			},
			args{
				file: "test",
			},
			"",
			true,
		},
		{
			"found command",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				file: "test",
			},
			"test",
			false,
		},
		{
			"found command file",
			fields{
				funcMap: map[string]CmdFunc{
					"/path/to/test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				file: "test",
			},
			"/path/to/test",
			false,
		},
		{
			"found file",
			fields{
				funcMap: map[string]CmdFunc{
					"/path/to/test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				file: "/path/to/test",
			},
			"/path/to/test",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				funcMap: tt.fields.funcMap,
			}
			got, err := e.LookPath(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FuncExec.LookPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("FuncExec.LookPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncExec_Command(t *testing.T) {
	type fields struct {
		funcMap map[string]CmdFunc
		envs    map[string]string
	}
	type args struct {
		name string
		arg  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *FuncCmd
	}{
		{
			"cmd with args",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"different path",
			fields{
				funcMap: map[string]CmdFunc{
					"/path/to/test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "/path/to/test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"lookpath failure",
			fields{
				funcMap: map[string]CmdFunc{},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				err:      errors.New(`exec: "test": executable file not found in $PATH`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				funcMap: tt.fields.funcMap,
				envs:    tt.fields.envs,
			}

			// set up tt.want with the correct fExec
			tt.want.fExec = e

			got := e.Command(tt.args.name, tt.args.arg...)

			// test err filrst since errors don't play nice with DeepEqual
			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
				got.(*FuncCmd).err = nil
			}
			if tt.want.Err() != nil {
				wantErr = tt.want.Err().Error()
				tt.want.err = nil
			}
			if gotErr != wantErr {
				t.Errorf("FuncExec.Command() got err %s, wanted err %s", gotErr, wantErr)
			}

			// check channels seperately as well
			if len(got.(*FuncCmd).exitCode) != len(tt.want.exitCode) ||
				cap(got.(*FuncCmd).exitCode) != cap(tt.want.exitCode) {
				t.Errorf("FuncExec.CommandContext() exitCode chan did not match what was expected")
			}
			if len(got.(*FuncCmd).ctxErr) != len(tt.want.ctxErr) ||
				cap(got.(*FuncCmd).ctxErr) != cap(tt.want.ctxErr) {
				t.Errorf("FuncExec.CommandContext() ctxErr chan did not match what was expected")
			}
			got.(*FuncCmd).exitCode = nil
			got.(*FuncCmd).ctxErr = nil
			tt.want.exitCode = nil
			tt.want.ctxErr = nil

			// now test the rest of the Cmd
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncExec.Command() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncExec_CommandContext(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	type fields struct {
		funcMap map[string]CmdFunc
		envs    map[string]string
	}
	type args struct {
		ctx  context.Context
		name string
		arg  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *FuncCmd
	}{
		{
			"cmd with args",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"different path",
			fields{
				funcMap: map[string]CmdFunc{
					"/path/to/test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "/path/to/test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"lookpath failure",
			fields{
				funcMap: map[string]CmdFunc{},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				err:      errors.New(`exec: "test": executable file not found in $PATH`),
			},
		},
		{
			"with timeout context",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
			},
			args{
				ctx:  timeoutCtx,
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      timeoutCtx,
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				funcMap: tt.fields.funcMap,
				envs:    tt.fields.envs,
			}

			// set up tt.want with the correct fExec
			tt.want.fExec = e

			got := e.CommandContext(tt.args.ctx, tt.args.name, tt.args.arg...)

			// test err filrst since errors don't play nice with DeepEqual
			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
				got.(*FuncCmd).err = nil
			}
			if tt.want.Err() != nil {
				wantErr = tt.want.Err().Error()
				tt.want.err = nil
			}
			if gotErr != wantErr {
				t.Errorf("FuncExec.CommandContext() got err %s, wanted err %s", gotErr, wantErr)
			}

			// check channels seperately as well
			if len(got.(*FuncCmd).exitCode) != len(tt.want.exitCode) ||
				cap(got.(*FuncCmd).exitCode) != cap(tt.want.exitCode) {
				t.Errorf("FuncExec.CommandContext() exitCode chan did not match what was expected")
			}
			if len(got.(*FuncCmd).ctxErr) != len(tt.want.ctxErr) ||
				cap(got.(*FuncCmd).ctxErr) != cap(tt.want.ctxErr) {
				t.Errorf("FuncExec.CommandContext() ctxErr chan did not match what was expected")
			}
			got.(*FuncCmd).exitCode = nil
			got.(*FuncCmd).ctxErr = nil
			tt.want.exitCode = nil
			tt.want.ctxErr = nil

			// now test the rest of the Cmd
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncExec.CommandContext() = \n%#v, want \n%#v", got, tt.want)
			}
		})
	}
}

func TestFuncCmd_Start(t *testing.T) {
	type fields struct {
		path    string
		stdout  io.Writer
		process *os.Process
		ctx     context.Context
		err     error
		fExec   *FuncExec
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty path",
			fields{
				path: "",
			},
			true,
		},
		{
			"lookpath err",
			fields{
				err: errors.New("lookpath error"),
			},
			true,
		},
		{
			"already run",
			fields{
				process: &os.Process{},
			},
			true,
		},
		{
			"context is already done",
			fields{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			true,
		},
		{
			"run cmd",
			fields{
				path: "test",
				fExec: &FuncExec{
					funcMap: map[string]CmdFunc{
						"test": func(fc *FuncCmd) int {
							fc.stdout.Write([]byte("test was run"))
							return 0
						},
					},
				},
				stdout: &bytes.Buffer{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:    tt.fields.path,
				stdout:  tt.fields.stdout,
				process: tt.fields.process,
				err:     tt.fields.err,
				fExec:   tt.fields.fExec,
			}
			if err := c.Start(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// check that process and pid are set
			if c.process == nil {
				t.Fatalf("FuncCmd.Start() start was run but process was not set")
			}

			if c.process.Pid == 0 {
				t.Fatalf("FuncCmd.Start() start was run but process id was not set")
			}

			// sleep for a short time to make sure the cmf func has time to run
			time.Sleep(time.Millisecond)

			// check that the function has been run (or at least started)
			gotStdout := c.stdout.(*bytes.Buffer).String()
			wantStdout := "test was run"
			if gotStdout != wantStdout {
				t.Errorf("FuncCmd.Start() cmd func was not run, got stdout %s but wanted %s", gotStdout, wantStdout)
			}
		})
	}
}

func TestFuncCmd_Wait(t *testing.T) {
	type fields struct {
		process      *os.Process
		processState *os.ProcessState
		ctx          context.Context
		ctxErr       chan error
		startErr     error
		exitCode     chan int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"not started",
			fields{
				process: nil,
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"start error",
			fields{
				startErr: errors.New("start error"),
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"wait already called",
			fields{
				process:      &os.Process{},
				processState: &os.ProcessState{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"process failed",
			fields{
				process: &os.Process{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 1 // non zero exit code
					return code
				}(),
			},
			true,
		},
		{
			"context already canceled",
			fields{
				process: &os.Process{},
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				ctxErr: func() chan error {
					err := make(chan error, 1)
					err <- errors.New("context canceled")
					return err
				}(),
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"success",
			fields{
				process: &os.Process{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				process:      tt.fields.process,
				processState: tt.fields.processState,
				ctx:          tt.fields.ctx,
				ctxErr:       tt.fields.ctxErr,
				startErr:     tt.fields.startErr,
				exitCode:     tt.fields.exitCode,
			}
			if err := c.Wait(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// check that process state was set
			if c.processState == nil {
				t.Errorf("FuncCmd.Wait() process state was not set")
			}
		})
	}
}

func TestFuncCmd_Run(t *testing.T) {
	type fields struct {
		path     string
		stdout   io.Writer
		ctx      context.Context
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		wantStart bool
	}{
		{
			"canceled command stdout locked",
			fields{
				path: "slow",
				ctx: func() context.Context {
					ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*5)
					return ctx
				}(),
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				stdout:   &bytes.Buffer{},
				fExec: &FuncExec{
					funcMap: map[string]CmdFunc{
						"slow": func(fc *FuncCmd) int {
							fc.stdout.Write([]byte("test was run"))
							time.Sleep(time.Second)
							fc.stdout.Write([]byte("sleep finished"))
							return 0
						},
					},
				},
			},
			true,
			true,
		},
		{
			"uncanceled command",
			fields{
				path: "fast",
				ctx: func() context.Context {
					ctx, _ := context.WithTimeout(context.Background(), time.Minute)
					return ctx
				}(),
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				stdout:   &bytes.Buffer{},
				fExec: &FuncExec{
					funcMap: map[string]CmdFunc{
						"fast": func(fc *FuncCmd) int {
							_, err := fc.stdout.Write([]byte("test was run"))
							if err != nil {
								return 1
							}
							return 0
						},
					},
				},
			},
			false,
			true,
		},
		{
			"missing command",
			fields{
				path:  "test",
				fExec: &FuncExec{},
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdout:   tt.fields.stdout,
				ctx:      tt.fields.ctx,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			if err := c.Run(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantStart {
				return
			}

			// check that the function was started
			gotStdout := c.stdout.(*bytes.Buffer).String()
			wantStdout := "test was run"
			if gotStdout != wantStdout {
				t.Errorf("FuncCmd.Run() cmd func was not run, got stdout %s but wanted %s", gotStdout, wantStdout)
			}
		})
	}
}
