package puffin

import (
	"bytes"
	"io"
)

type lockableBuf struct {
	read  io.Reader
	write io.Writer
	lock  bool
}

// Read reads data into the in reader and also the out writer if it has a read method
func (b *lockableBuf) Read(p []byte) (int, error) {
	if b.lock {
		return len(p), nil
	}

	if w, ok := b.write.(io.Reader); ok {
		return w.Read(p)
	}

	return b.read.Read(p)
}

// Write writes data into the out writer and also the in reader if it has a write method
func (b *lockableBuf) Write(p []byte) (int, error) {
	if b.lock {
		return len(p), nil
	}

	if r, ok := b.read.(io.Writer); ok {
		return r.Write(p)
	}

	return b.write.Write(p)
}

// Close closes the interal io.Reader and io.Writer if they are not nil
// and if they have a close function
func (b *lockableBuf) Close() error {
	if b.read != nil {
		if c, ok := b.read.(io.ReadCloser); ok {
			if err := c.Close(); err != nil {
				return err
			}
		}
	}
	if b.write != nil {
		if c, ok := b.write.(io.WriteCloser); ok {
			if err := c.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Bytes returns the bytes in the reader or writer
func (b *lockableBuf) Bytes() []byte {
	buf := &bytes.Buffer{}

	if b.write != nil {
		if r, ok := b.write.(io.Reader); ok {
			io.Copy(buf, r)
		}
	}

	if b.read != nil {
		io.Copy(buf, b.read)
	}

	return buf.Bytes()
}
