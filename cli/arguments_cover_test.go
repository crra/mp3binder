package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/stretchr/testify/assert"
)

const (
	validCoverFile1 = "cover.jpg"
	validCoverFile2 = "cover.jpeg"
	validCoverFile3 = "cover.png"

	invalidCoverFile = "_"
)

func TestNonExistingCoverFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.coverFile = invalidCoverFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInvalidCoverFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1, validFileName2, invalidCoverFile)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.coverFile = invalidCoverFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestCoverFileIsDir(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()

	_ = makeEmptyFiles(fs, root, validFileName1, validFileName2)
	fs.MkdirAll(filepath.Join(root, validCoverFile1), 0o755)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.coverFile = validCoverFile1

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestValidCoverFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	for _, f := range []string{
		validCoverFile1,
		validCoverFile2,
		validCoverFile3,
		strings.ToUpper(validCoverFile1),
		strings.ToUpper(validCoverFile2),
		strings.ToUpper(validCoverFile3),
	} {
		mediaFile := makeEmptyFiles(fs, root, f)

		a := newDefaultApplication(aferox.NewAferox(root, fs))
		a.coverFile = mediaFile[0]

		err := a.args(nil, []string{"."})
		assert.NoError(t, err)
	}
}

func TestDiscoverCoverFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	coverFiles := makeEmptyFiles(fs, root, validCoverFile1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, coverFiles[0], a.coverFile)
	}
}

func TestNoCoverFileDiscovery(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	_ = makeEmptyFiles(fs, root, validCoverFile1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.noDiscovery = true

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "", a.coverFile)
	}
}

func TestDiscoverCoverFileUppercased(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	coverFiles := makeEmptyFiles(fs, root, strings.ToUpper(validCoverFile1))

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, coverFiles[0], a.coverFile)
	}
}
