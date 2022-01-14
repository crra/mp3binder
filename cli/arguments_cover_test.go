package cli

import (
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
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
	t.Parallel()
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
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	fs.MkdirAll("/"+validCoverFile1, 0755)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		coverFile: validCoverFile1,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestValidCoverFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	for _, f := range []string{
		validCoverFile1,
		validCoverFile2,
		validCoverFile3,
		strings.ToUpper(validCoverFile1),
		strings.ToUpper(validCoverFile2),
		strings.ToUpper(validCoverFile3),
	} {
		afero.WriteFile(fs, "/"+f, []byte("cover"), 0644)

		a := &application{
			fs:        aferox.NewAferox("/", fs),
			coverFile: f,
		}

		err := a.args(nil, []string{"."})
		assert.NoError(t, err)
	}
}

func TestDiscoverCoverFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validCoverFile1, []byte("cover"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validCoverFile1, a.coverFile)
	}
}

func TestNoCoverFileDiscovery(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validCoverFile1, []byte("cover"), 0644)

	a := &application{
		fs:          aferox.NewAferox("/", fs),
		noDiscovery: true,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "", a.coverFile)
	}
}

func TestDiscoverCoverFileUppercased(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+strings.ToUpper(validCoverFile1), []byte("cover"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+strings.ToUpper(validCoverFile1), a.coverFile)
	}
}
