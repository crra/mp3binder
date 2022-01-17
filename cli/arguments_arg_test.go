package cli

import (
	"fmt"
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

	validInterlaceFile1 = "interlace.mp3"
	validInterlaceFile2 = "_interlace.mp3"
)

var defaultFileContent = []byte{}

func filesToMediaFiles(files []string, path string) []string {
	mediaFiles := make([]string, len(files))
	for i, f := range files {
		mediaFiles[i] = path + f
	}

	return mediaFiles
}

func makeFiles(fs afero.Fs, files []string, path string) []string {
	for _, f := range files {
		afero.WriteFile(fs, path+f, defaultFileContent, 0644)
	}

	return filesToMediaFiles(files, path)
}

func withTwoValidFiles(fs afero.Fs, path string) []string {
	return makeFiles(fs, []string{validFileName1, validFileName2}, path)
}

func withThreeValidFiles(fs afero.Fs, path string) []string {
	return makeFiles(fs, []string{validFileName1, validFileName2, validFileName3}, path)
}

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
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDirectoryWithNoValidFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+invalidFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+invalidFileName2, defaultFileContent, 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithOneFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)

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
			mediaFiles := makeFiles(fs, f, "/")

			a := &application{
				fs:        aferox.NewAferox("/", fs),
				overwrite: true,
			}

			err := a.args(nil, []string{"."})
			if assert.NoError(t, err) {
				assert.Equal(t, mediaFiles, a.mediaFiles)
			}
		})
	}
}

func TestNoParametersDefaultsToDirectory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	mediaFiles := withTwoValidFiles(fs, "/")

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestDirectoryWithTwoFilesAndExplicitlyUsedMagicInterlaceFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validInterlaceFile2, defaultFileContent, 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	files := []string{validFileName1, "interlace.mp3", validFileName2, "interlace.mp3", "_interlace.mp3"}
	mediaFiles := filesToMediaFiles(files, "/")

	err := a.args(nil, files)
	if assert.NoError(t, err) {
		assert.Equal(t, len(files), len(a.mediaFiles))
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestSubDirectoryWithTwoFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	withTwoValidFiles(fs, "/"+sampleDirectory+"/")

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestOneFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestOneInvalidFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+invalidFileName1, defaultFileContent, 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{invalidFileName1})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestTwoFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	mediaFiles := withTwoValidFiles(fs, "/")

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
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
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	fs.MkdirAll("/"+validFileName2, 0755)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.Error(t, err, ErrAtLeastTwo)
}

func TestFilesAndDirectoryNoDuplicates(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	mediaFiles := withTwoValidFiles(fs, "/")

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{".", validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryDirectoryFirst(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName3, defaultFileContent, 0644)

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
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName3, defaultFileContent, 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{validFileName2, validFileName1, "."})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName2,
			"/" + validFileName1,
			"/" + validFileName3,
		}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryFirst(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName3, defaultFileContent, 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{".", validFileName2, validFileName1})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName3,
			"/" + validFileName2,
			"/" + validFileName1,
		}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryFirstAndLast(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName3, defaultFileContent, 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{".", validFileName2, validFileName1, "."})
	if assert.NoError(t, err) {
		assert.Equal(t, []string{
			"/" + validFileName3,
			"/" + validFileName2,
			"/" + validFileName1,
		}, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueButKeepDuplicatesFromArg(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)

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
