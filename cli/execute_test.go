package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/stretchr/testify/assert"
)

type testCollector struct {
	err error

	parent    context.Context
	output    io.WriteSeeker
	audioOnly io.ReadWriteSeeker
	input     []io.Reader
	options   []mp3binder.Option
}

func (t *testCollector) Bind(parent context.Context, output io.WriteSeeker, audioOnly io.ReadWriteSeeker, input []io.Reader, options ...any) error {
	t.parent = parent
	t.output = output
	t.audioOnly = audioOnly
	t.input = input
	// t.options = options

	return t.err
}

func TestCreateEmptyFile(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	root, fs := newTestFilesystem()

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.binder = tc
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		_, err := fs.Stat(a.outputPath)
		assert.NoError(t, err)
	}
}

func TestRemoveOutputFileOnError(t *testing.T) {
	t.Parallel()
	tc := &testCollector{err: assert.AnError}
	root, fs := newTestFilesystem()

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.binder = tc
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.run(nil, nil)
	if assert.Error(t, err) {
		_, err := fs.Stat(a.outputPath)
		assert.ErrorIs(t, err, os.ErrNotExist)
	}
}

func TestMediaFiles(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	root, fs := newTestFilesystem()
	mediaFiles := withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.binder = tc
	a.mediaFiles = mediaFiles
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, len(mediaFiles), len(tc.input))
	}
}

func TestInterlaceWithTwo(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	root, fs := newTestFilesystem()
	mediaFiles := withTwoValidFiles(fs, root)
	_ = makeFiles(fs, root, validInterlaceFile1, validInterlaceFile2)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	a.binder = tc
	a.mediaFiles = mediaFiles
	a.interlaceFile = filepath.Join(root, validInterlaceFile1)
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, filepathJoin(root, validFileName1, validInterlaceFile1, validFileName2), a.mediaFiles)
		assert.Equal(t, len(a.mediaFiles), len(tc.input))
	}
}

func TestInterlaceWithThree(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	root, fs := newTestFilesystem()
	mediaFiles := withThreeValidFiles(fs, root)
	_ = makeFiles(fs, root, validInterlaceFile1, validInterlaceFile2)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.binder = tc
	a.mediaFiles = mediaFiles
	a.interlaceFile = filepath.Join(root, validInterlaceFile1)
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, filepathJoin(root, validFileName1, validInterlaceFile1, validFileName2, validInterlaceFile1, validFileName3), a.mediaFiles)
		assert.Equal(t, len(a.mediaFiles), len(tc.input))
	}
}
