package ioext

import (
	"io"
	"sync"
)

// https://github.com/golang/go/issues/25408

type onceCloser struct {
	c    io.Closer
	once sync.Once
	err  error
}

// OnceCloser returns a Closer wrapping c that guarantees it only calls c.Close
// once and is safe for use by multiple goroutines. Each call to the returned Closer
// will return the same value, as returned by c.Close.
func OnceCloser(c io.Closer) io.Closer {
	return &onceCloser{c: c}
}

func (c *onceCloser) Close() error {
	c.once.Do(c.close)
	return c.err
}

func (c *onceCloser) close() {
	c.err = c.c.Close()
}
