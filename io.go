package puffin

import (
	"bytes"
	"io"
)

// lockableBuff is a io.ReadWriter where the Read and Write methods can be locked
// to prevent further updates to the underlying buffers
type lockableBuffer struct {
	reader      io.Reader
	writer      io.Writer
	writeLocked bool
	readLocked  bool
}

func newLockableBuffer() *lockableBuffer {
	buf := &bytes.Buffer{}
	return &lockableBuffer{
		reader: buf,
		writer: buf,
	}
}

func newLockableReader(reader io.Reader) *lockableBuffer {
	return &lockableBuffer{
		reader:      reader,
		writeLocked: true,
	}
}

func newLockableWriter(writer io.Writer) *lockableBuffer {
	return &lockableBuffer{
		writer:     writer,
		readLocked: true,
	}
}

// Write is a passthrough to the underlying io.ReadWriter's Write method.
// If writing is locked however, it becomes a no-op
func (rw *lockableBuffer) Write(p []byte) (n int, err error) {
	if rw.writeLocked {
		return len(p), nil
	}

	return rw.writer.Write(p)
}

// Read is a passthrough to the underlying io.ReadWriter's Read method.
// If reading is locked however, it becomes a no-op
func (rw *lockableBuffer) Read(p []byte) (n int, err error) {
	if rw.readLocked {
		return 0, nil
	}

	return rw.reader.Read(p)
}

// Close will close the underlying io.Read and io.Writer if it has a close method as well
// otherwise it's a no-op
func (rw *lockableBuffer) Close() error {
	if closer, ok := rw.reader.(io.Closer); ok {
		return closer.Close()
	}
	if closer, ok := rw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// LockWrite locks writing to the buffer, preventing future writes to the underlying buffer
func (rw *lockableBuffer) LockWrite() {
	rw.writeLocked = true
}

// LockRead locks reading from the buffer, preventing future reads from the underlying buffer
func (rw *lockableBuffer) LockRead() {
	rw.readLocked = true
}

// Bytes reads the contents out of the buffer and returns is as a byte slice
func (rw *lockableBuffer) Bytes() []byte {
	if rw.reader == nil {
		return nil
	}

	p, _ := io.ReadAll(rw.reader)
	return p
}

// String reads the contents out of the buffer and returns it as a string
func (rw *lockableBuffer) String() string {
	if rw.reader == nil {
		return ""
	}

	p, _ := io.ReadAll(rw.reader)
	return string(p)
}
