package cli

import (
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	validFileName1 = "sample1.mp3"
	validFileName2 = "sample2.mp3"

	invalidFileName1 = "sample1.mp33"
	invalidFileName2 = "sample2.mp33"
)

func TestNoParameters(t *testing.T) {
	a := &application{
		fs: aferox.NewAferox("/", afero.NewMemMapFs()),
	}

	err := a.args(nil, []string{})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithNoFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithNoValidFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+invalidFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+invalidFileName2, []byte("2"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithOneFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestDirectoryWithTwoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.NoError(t, err)
}

func TestOneFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestOneInvalidFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+invalidFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{invalidFileName1})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestTwoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestOneFileAndADirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	fs.MkdirAll("/"+validFileName2, 0755)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.Error(t, err, ErrAtLeastTwo)
}
