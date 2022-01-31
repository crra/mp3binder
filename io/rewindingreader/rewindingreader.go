// package rollingreader wraps a ReadSeeker and exposes it as a simple Reader. When the end of the stream is reached,
// the read cursor is reset to the beginning (rewound) and another consumer can read the stream till it's end.
package rewindingreader

import (
	"errors"
	"io"
)

type rewindingReader struct {
	io.Reader

	rewind     bool
	readSeeker io.ReadSeeker
}

// New creates a new io.Reader rewinds itself to the beginning once the end of the stream is reached.
func New(rs io.ReadSeeker) io.Reader {
	return &rewindingReader{readSeeker: rs}
}

// Read implements the io.Reader interface.
func (r *rewindingReader) Read(p []byte) (int, error) {
	if r.rewind {
		if _, err := r.readSeeker.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}

		r.rewind = false
	}

	n, err := r.readSeeker.Read(p)

	l := len(p)
	if (l > 0 && n < l) || (err != nil && errors.Is(err, io.EOF)) {
		r.rewind = true
	}

	return n, err
}
