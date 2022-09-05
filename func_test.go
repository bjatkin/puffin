package puffin

import (
	"context"
	"errors"
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
		want   Cmd
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
				path: "test",
				args: []string{"test", "arg1", "arg2"},
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
				path: "/path/to/test",
				args: []string{"test", "arg1", "arg2"},
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
				path: "test",
				args: []string{"test", "arg1", "arg2"},
				err:  errors.New(`exec: "test": executable file not found in $PATH, wanted err`),
			},
		},
		{
			"with environ",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
				envs: map[string]string{
					"TEST":  "true",
					"DEBUG": "false",
				},
			},
			args{
				name: "test",
				arg:  []string{"args1", "arg2"},
			},
			&FuncCmd{
				path: "test",
				args: []string{"test", "args1", "arg2"},
				env: map[string]string{
					"TEST":  "true",
					"DEBUG": "false",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				funcMap: tt.fields.funcMap,
				envs:    tt.fields.envs,
			}

			got := e.Command(tt.args.name, tt.args.arg...)

			if !reflect.DeepEqual(got.Args(), tt.want.Args()) {
				t.Errorf("FuncExec.Command() got args %v, wanted args %v", got.Args(), tt.want.Args())
			}

			if got.Path() != tt.want.Path() {
				t.Errorf("FuncExec.Command() got path %v, wanted path %v", got.Path(), tt.want.Path())
			}

			if !reflect.DeepEqual(got.Environ(), tt.want.Environ()) {
				t.Errorf("FuncExec.Command() got environ %v, wanted environ %v", got.Environ(), tt.want.Environ())
			}

			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
			}
			if tt.want.Err() != nil {
				wantErr = got.Err().Error()
			}

			if gotErr != wantErr {
				t.Errorf("FuncExec.Command() got err %v, wanted err %v", gotErr, wantErr)
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
		want   Cmd
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
				ctx:  context.Background(),
				path: "test",
				args: []string{"test", "arg1", "arg2"},
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
				ctx:  context.Background(),
				path: "/path/to/test",
				args: []string{"test", "arg1", "arg2"},
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
				ctx:  context.Background(),
				path: "test",
				args: []string{"test", "arg1", "arg2"},
				err:  errors.New(`exec: "test": executable file not found in $PATH, wanted err`),
			},
		},
		{
			"with environ",
			fields{
				funcMap: map[string]CmdFunc{
					"test": func(fc *FuncCmd) int { return 0 },
				},
				envs: map[string]string{
					"TEST":  "true",
					"DEBUG": "false",
				},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"args1", "arg2"},
			},
			&FuncCmd{
				ctx:  context.Background(),
				path: "test",
				args: []string{"test", "args1", "arg2"},
				env: map[string]string{
					"TEST":  "true",
					"DEBUG": "false",
				},
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
				ctx:  timeoutCtx,
				path: "test",
				args: []string{"test", "arg1", "arg2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				funcMap: tt.fields.funcMap,
				envs:    tt.fields.envs,
			}

			got := e.CommandContext(tt.args.ctx, tt.args.name, tt.args.arg...)

			gotCtx := got.(*FuncCmd).ctx
			wantCtx := tt.want.(*FuncCmd).ctx
			if !reflect.DeepEqual(gotCtx, wantCtx) {
				t.Errorf("FuncExec.CommandContext() got context %v, wanted context %v", gotCtx, wantCtx)
			}

			if !reflect.DeepEqual(got.Args(), tt.want.Args()) {
				t.Errorf("FuncExec.CommandContext() got args %v, wanted args %v", got.Args(), tt.want.Args())
			}

			if got.Path() != tt.want.Path() {
				t.Errorf("FuncExec.CommandContext() got path %v, wanted path %v", got.Path(), tt.want.Path())
			}

			if !reflect.DeepEqual(got.Environ(), tt.want.Environ()) {
				t.Errorf("FuncExec.CommandContext() got environ %v, wanted environ %v", got.Environ(), tt.want.Environ())
			}

			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
			}
			if tt.want.Err() != nil {
				wantErr = got.Err().Error()
			}

			if gotErr != wantErr {
				t.Errorf("FuncExec.CommandContext() got err %v, wanted err %v", gotErr, wantErr)
			}
		})
	}
}
