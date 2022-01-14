package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	sampleDirectory = "sampleDirectory"

	validFileName1 = "validSampleFile1.mp3"
	validFileName2 = "validSampleFile2.mp3"
	validFileName3 = "validSampleFile3.mp3"

	invalidFileName1 = "invalidSampleFile1.mp33"
	invalidFileName2 = "invalidSampleFile2.mp33"
)

func TestDirectoryWithNoFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithNonExistingDirectory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{sampleDirectory})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDirectoryWithNonExistingFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDirectoryWithNoValidFile(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestDirectoryWithTwoFiles(t *testing.T) {
	t.Parallel()
	for i, f := range [][]string{
		{
			validFileName1,
			validFileName2,
		},
		{
			strings.ToUpper(validFileName1),
			strings.ToUpper(validFileName2),
		},
	} {
		f := f // pin
		t.Run(fmt.Sprintf("Index-%d", i), func(t *testing.T) {
			t.Parallel()
			fs := afero.NewMemMapFs()
			for _, n := range f {
				afero.WriteFile(fs, "/"+n, []byte(""), 0644)
			}

			a := &application{
				fs:        aferox.NewAferox("/", fs),
				overwrite: true,
			}

			err := a.args(nil, []string{"."})
			assert.NoError(t, err)
		})
	}
}

func TestNoParametersDefaultsToDirectory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte(""), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte(""), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{})
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(a.mediaFiles))
	}
}

func TestDirectoryWithTwoFilesAndIgnoredMagicInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/interlace.mp3", []byte("interlace"), 0644)
	afero.WriteFile(fs, "/_interlace.mp3", []byte("interlace"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(a.mediaFiles))
	}
}

func TestDirectoryWithTwoFilesAndExplicitlyUsedMagicInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/interlace.mp3", []byte("interlace"), 0644)
	afero.WriteFile(fs, "/_interlace.mp3", []byte("interlace"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	files := []string{validFileName1, "interlace.mp3", validFileName2, "interlace.mp3", "_interlace.mp3"}
	err := a.args(nil, files)
	if assert.NoError(t, err) {
		assert.Equal(t, len(files), len(a.mediaFiles))
	}
}

func TestSubDirectoryWithTwoFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName1), []byte("1"), 0644)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName2), []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestOneFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestOneInvalidFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+invalidFileName1, []byte("1"), 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{invalidFileName1})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestTwoFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestOneFileAndDirectory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	fs.MkdirAll("/"+validFileName2, 0755)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.Error(t, err, ErrAtLeastTwo)
}

func TestFilesAndDirectoryUnique(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{".", validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"/" + validFileName1, "/" + validFileName2}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryDirectoryFirst(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validFileName3, []byte("3"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{".", validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName3,
			"/" + validFileName1,
			"/" + validFileName2,
		}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryLast(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validFileName3, []byte("3"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{validFileName1, validFileName2, "."})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName1,
			"/" + validFileName2,
			"/" + validFileName3,
		}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueButKeepDuplicatesFromArg(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{
		".",
		validFileName1,
		validFileName1,
		validFileName2,
		validFileName2,
	})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName1,
			"/" + validFileName1,
			"/" + validFileName2,
			"/" + validFileName2,
		}, a.mediaFiles)
	}
}
