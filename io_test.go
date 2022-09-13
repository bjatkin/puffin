package puffin

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func Test_lockableBuffer_Write(t *testing.T) {
	type fields struct {
		writer      io.Writer
		writeLocked bool
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantN     int
		wantErr   bool
		wantBytes []byte
	}{
		{
			"successful writer",
			fields{
				writer: &bytes.Buffer{},
			},
			args{
				p: []byte("successful write"),
			},
			16,
			false,
			[]byte("successful write"),
		},
		{
			"write locked",
			fields{
				writer:      &bytes.Buffer{},
				writeLocked: true,
			},
			args{
				p: []byte("locked write"),
			},
			12,
			false,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &lockableBuffer{
				writer:      tt.fields.writer,
				writeLocked: tt.fields.writeLocked,
			}
			gotN, err := rw.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("lockableBuffer.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("lockableBuffer.Write() = %v, want %v", gotN, tt.wantN)
			}

			gotBytes := tt.fields.writer.(*bytes.Buffer).Bytes()
			if !reflect.DeepEqual(gotBytes, tt.wantBytes) {
				t.Errorf("lockableBuffer.Write() bytes = %v, wantBytes %v", gotBytes, tt.wantBytes)
			}
		})
	}
}

func Test_lockableBuffer_Read(t *testing.T) {
	type fields struct {
		reader     io.Reader
		readLocked bool
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantN     int
		wantErr   bool
		wantBytes []byte
	}{
		{
			"successful read",
			fields{
				reader: strings.NewReader("successful read"),
			},
			args{
				p: make([]byte, 15),
			},
			15,
			false,
			[]byte("successful read"),
		},
		{
			"locked read",
			fields{
				reader:     strings.NewReader("locked read"),
				readLocked: true,
			},
			args{
				p: make([]byte, 100),
			},
			0,
			false,
			make([]byte, 100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &lockableBuffer{
				reader:     tt.fields.reader,
				readLocked: tt.fields.readLocked,
			}
			gotN, err := rw.Read(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("lockableBuffer.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("lockableBuffer.Read() = %v, want %v", gotN, tt.wantN)
			}

			if !reflect.DeepEqual(tt.args.p, tt.wantBytes) {
				t.Errorf("lockableBuffer.Write() bytes = %v, wantBytes %v", tt.args.p, tt.wantBytes)
			}
		})
	}
}
