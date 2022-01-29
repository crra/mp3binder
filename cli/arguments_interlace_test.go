package cli

// TODO: this is a direct copy of the cover file test cases
//       the code is already generic, but the tests are not.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	invalidInterlaceFile = "_"
)

func TestNonExistingInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.interlaceFile = invalidInterlaceFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestInvalidInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+invalidInterlaceFile, defaultFileContent, 0644)

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.interlaceFile = invalidInterlaceFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestInterlaceFileIsDir(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	fs.MkdirAll("/"+validInterlaceFile1, 0755)

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.interlaceFile = validInterlaceFile1

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

			fs := afero.NewMemMapFs()
			afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
			afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
			afero.WriteFile(fs, "/"+f, defaultFileContent, 0644)

			a := newDefaultApplication(aferox.NewAferox("/", fs))
			a.interlaceFile = f

			err := a.args(nil, []string{"."})
			assert.NoError(t, err)
		})
	}
}

func TestDiscoverInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile1, defaultFileContent, 0644)

	a := newDefaultApplication(aferox.NewAferox("/", fs))

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validInterlaceFile1, a.interlaceFile)
	}
}

func TestNoInterlaceFileDiscovery(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile1, defaultFileContent, 0644)

	a := newDefaultApplication(aferox.NewAferox("/", fs))
	a.noDiscovery = true

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "", a.interlaceFile)
	}
}
