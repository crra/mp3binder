package cli

import (
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	validCoverFile   = "cover.jpg"
	invalidCoverFile = "_"
)

func TestNonExistingCoverFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		coverFile: invalidCoverFile,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInvalidCoverFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+invalidCoverFile, []byte("cover"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		coverFile: invalidCoverFile,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestCoverFileIsDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	fs.MkdirAll("/"+validCoverFile, 0755)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		coverFile: validCoverFile,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestValidCoverFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validCoverFile, []byte("cover"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		coverFile: validCoverFile,
		overwrite: true,
	}

	err := a.args(nil, []string{"."})
	assert.NoError(t, err)
}

func TestDiscoverCoverFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validCoverFile, []byte("cover"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validCoverFile, a.coverFile)
	}
}

func TestNoCoverFileDiscovery(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validCoverFile, []byte("cover"), 0644)

	a := &application{
		fs:          aferox.NewAferox("/", fs),
		noDiscovery: true,
		overwrite:   true,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "", a.coverFile)
	}
}
