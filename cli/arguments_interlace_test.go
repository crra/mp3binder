package cli

// TODO: this is a direct copy of the cover file test cases
//       the code is already generic, but the tests are not.

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/stretchr/testify/assert"
)

const (
	invalidInterlaceFile = "_"
)

func TestNonExistingInterlaceFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.interlaceFile = invalidInterlaceFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInvalidInterlaceFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	invalidInterlaceFiles := makeEmptyFiles(fs, root, invalidInterlaceFile)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.interlaceFile = invalidInterlaceFiles[0]

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestInterlaceFileIsDir(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	invalidInterlaceFileDirPath := filepath.Join(root, validInterlaceFile1)
	fs.MkdirAll(invalidInterlaceFileDirPath, 0o755)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.interlaceFile = invalidInterlaceFileDirPath

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestValidInterlaceFile(t *testing.T) {
	t.Parallel()
	for i, f := range []string{
		validInterlaceFile1,
		strings.ToUpper(validInterlaceFile1),
	} {
		f := f // pin
		t.Run(fmt.Sprintf("Index-%d", i), func(t *testing.T) {
			t.Parallel()

			root, fs := newTestFilesystem()
			_ = withTwoValidFiles(fs, root)
			interlaceFiles := makeEmptyFiles(fs, root, f)

			a := newDefaultApplication(aferox.NewAferox(root, fs))
			a.interlaceFile = interlaceFiles[0]

			err := a.args(nil, []string{"."})
			assert.NoError(t, err)
		})
	}
}

func TestDiscoverInterlaceFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	interlaceFiles := makeEmptyFiles(fs, root, validInterlaceFile1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, interlaceFiles[0], a.interlaceFile)
	}
}

func TestNoInterlaceFileDiscovery(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	_ = makeEmptyFiles(fs, root, validInterlaceFile1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.noDiscovery = true

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "", a.interlaceFile)
	}
}
