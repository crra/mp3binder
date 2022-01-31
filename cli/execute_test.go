package cli

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/spf13/afero"
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
	fs := afero.NewMemMapFs()

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.binder = tc
	a.outputPath = "/" + validOutputFile

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		_, err := fs.Stat(a.outputPath)
		assert.NoError(t, err)
	}
}

func TestRemoveOutputFileOnError(t *testing.T) {
	t.Parallel()
	tc := &testCollector{err: assert.AnError}
	fs := afero.NewMemMapFs()

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.binder = tc
	a.outputPath = "/" + validOutputFile

	err := a.run(nil, nil)
	if assert.Error(t, err) {
		_, err := fs.Stat(a.outputPath)
		assert.ErrorIs(t, err, os.ErrNotExist)
	}
}

func TestMediaFiles(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	fs := afero.NewMemMapFs()
	mediaFiles := withTwoValidFiles(fs, "/")

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.binder = tc
	a.mediaFiles = mediaFiles
	a.outputPath = "/" + validOutputFile

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, len(mediaFiles), len(tc.input))
	}
}

func TestInterlaceWithTwo(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	fs := afero.NewMemMapFs()
	mediaFiles := withTwoValidFiles(fs, "/")
	afero.WriteFile(fs, "/"+validInterlaceFile1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile2, defaultFileContent, 0644)

	expected := []string{
		"/" + validFileName1,
		"/" + validInterlaceFile1,
		"/" + validFileName2,
	}

	a := newDefaultApplication(aferox.NewAferox("/", fs))

	a.binder = tc
	a.mediaFiles = mediaFiles
	a.interlaceFile = "/" + validInterlaceFile1
	a.outputPath = "/" + validOutputFile

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, a.mediaFiles)
		assert.Equal(t, len(a.mediaFiles), len(tc.input))
	}
}

func TestInterlaceWithThree(t *testing.T) {
	t.Parallel()
	tc := &testCollector{}
	fs := afero.NewMemMapFs()
	mediaFiles := withThreeValidFiles(fs, "/")
	afero.WriteFile(fs, "/"+validInterlaceFile1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile2, defaultFileContent, 0644)

	expected := []string{
		"/" + validFileName1,
		"/" + validInterlaceFile1,
		"/" + validFileName2,
		"/" + validInterlaceFile1,
		"/" + validFileName3,
	}

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.binder = tc
	a.mediaFiles = mediaFiles
	a.interlaceFile = "/" + validInterlaceFile1
	a.outputPath = "/" + validOutputFile

	err := a.run(nil, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, a.mediaFiles)
		assert.Equal(t, len(a.mediaFiles), len(tc.input))
	}
}