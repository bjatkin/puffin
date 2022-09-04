package puffin

import (
	"io"
)

// lockableWriter is a io.Writer where the write method can be locked
// to prevent further writing to the buffer
type lockableWriter struct {
	io.Writer
	locked bool
}

// Write is a passthrough to the underlying io.Write unless the
// struct is locked, in which case this function becomes a no op
func (w *lockableWriter) Write(p []byte) (n int, err error) {
	if w.locked {
		return len(p), nil
	}
	return w.Writer.Write(p)
}

// Lock locks the writer to prevent future writes to the underlying io.Writer
func (w *lockableWriter) Lock() {
	w.locked = true
}

// lockableReader is an io.Reader where the read method can be locked
// to prevent further reading from the buffer
type lockableReader struct {
	io.Reader
	locked bool
}

// Read is a passthrough to the underlying io.Reader unless the
// struct is locked, in which case this function becomes a no op
func (r *lockableReader) Read(p []byte) (n int, err error) {
	if r.locked {
		return 0, nil
	}

	return r.Reader.Read(p)
}

// Lock locks the reader to prevent future reads from the underlying io.Reader
func (r *lockableReader) Lock() {
	r.locked = true
}

// lockableBuff is a io.ReadWriter where the Read and Write methods can be locked
// to prevent further updates to the underlying buffer
type lockableBuffer struct {
	io.ReadWriter
	writeLocked bool
	readLocked  bool
}

// Write is a passthrough to the underlying io.ReadWriter's Write method.
// If writing is locked however, it becomes a no-op
func (rw *lockableBuffer) Write(p []byte) (n int, err error) {
	if rw.writeLocked {
		return len(p), nil
	}

	return rw.ReadWriter.Write(p)
}

// Read is a passthrough to the underlying io.ReadWriter's Read method.
// If reading is locked however, it becomes a no-op
func (rw *lockableBuffer) Read(p []byte) (n int, err error) {
	if rw.readLocked {
		return 0, nil
	}

	return rw.ReadWriter.Read(p)
}

// Close will close the underlying io.ReadWriter if it has a close method as well
// otherwise it's a no-op
func (rw *lockableBuffer) Close() error {
	if closer, ok := rw.ReadWriter.(io.Closer); ok {
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
	p, _ := io.ReadAll(rw.ReadWriter)
	return p
}
