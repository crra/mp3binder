package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	inputFile       = "input.txt"
	nonExistingFile = "_"
)

func TestInputFileIsNotExisting(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, nil)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInputFileIsDirectory(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	err := fs.Mkdir(filepath.Join(afero.FilePathSeparator, inputFile), 0o755)
	if assert.NoError(t, err) {
		a := newDefaultApplication(aferox.NewAferox(root, fs))
		a.inputFile = inputFile

		err := a.args(nil, nil)
		assert.ErrorIs(t, err, ErrInvalidFile)
	}
}

func TestInputFileIsIsEmpty(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, inputFile)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, nil)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestInputFileNonExistingFiles(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()

	b := &bytes.Buffer{}
	b.WriteString(nonExistingFile)
	afero.WriteFile(fs, filepath.Join(root, inputFile), b.Bytes(), 0o644)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, nil)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInputFileOnlyOneFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1)

	b := &bytes.Buffer{}
	b.WriteString(validFileName1)
	afero.WriteFile(fs, filepath.Join(root, inputFile), b.Bytes(), 0o644)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, nil)
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestInputFileWithTwoFiles(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1, validFileName2)

	b := &bytes.Buffer{}
	b.WriteString(validFileName1)
	b.WriteString("\n")
	b.WriteString(validFileName2)
	afero.WriteFile(fs, filepath.Join(root, inputFile), b.Bytes(), 0o644)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, nil)
	assert.NoError(t, err)
}

func TestInputFileWithOneFileAndAndArgument(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFilesOrdered := makeEmptyFiles(fs, root, validFileName1, validFileName2)

	b := &bytes.Buffer{}
	b.WriteString(validFileName2)
	afero.WriteFile(fs, filepath.Join(root, inputFile), b.Bytes(), 0o644)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.inputFile = inputFile

	err := a.args(nil, []string{validFileName1})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFilesOrdered, a.mediaFiles)
	}
}
