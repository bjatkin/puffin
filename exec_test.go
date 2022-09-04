package puffin

import "testing"

func TestExec_LookPath(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		funcs   map[string]CmdFunc
		want    string
		wantErr bool
	}{
		{
			"missing func",
			args{
				file: "test",
			},
			map[string]CmdFunc{},
			"",
			true,
		},
		{
			"found command",
			args{
				file: "test",
			},
			map[string]CmdFunc{
				"test": func(fc *FuncCmd) int { return 0 },
			},
			"test",
			false,
		},
		{
			"found command file",
			args{
				file: "test",
			},
			map[string]CmdFunc{
				"/path/to/test": func(fc *FuncCmd) int { return 0 },
			},
			"/path/to/test",
			false,
		},
		{
			"found file",
			args{
				file: "/path/to/test",
			},
			map[string]CmdFunc{
				"/path/to/test": func(fc *FuncCmd) int { return 0 },
			},
			"/path/to/test",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewFuncExec(WithFuncMap(tt.funcs))
			got, err := exec.LookPath(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FuncExec.LookPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("FuncExec.LookPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
