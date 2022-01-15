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
	validOutputFile = "output.mp3"
)

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
}

func TestOutputFileNotExisting(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputPath: validOutputFile,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validOutputFile, a.outputPath)
	}
}

func TestOutputFileMissingExtension(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputPath: fileNameWithoutExtension(validOutputFile),
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validOutputFile, a.outputPath)
	}
}

func TestOutputFileExistingNoOverwrite(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validOutputFile, []byte("out"), 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputPath: validOutputFile,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrOutputFileExists)
}

func TestOutputFileExistingOverwrite(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validFileName2, defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+validOutputFile, []byte("out"), 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputPath: validOutputFile,
		overwrite:  true,
	}

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, "/"+validOutputFile, a.outputPath)
	}
}

func TestAsOutputFile(t *testing.T) {
	t.Parallel()
	const (
		file      = "file"
		extension = ".mp3"
		fullName  = file + extension
	)

	assert.Equal(t, fullName, asOutputFile(file))
	assert.Equal(t, fullName, asOutputFile(fullName))
}

func TestOutputFileFromSampleDirectory1(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName1), defaultFileContent, 0644)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName2), defaultFileContent, 0644)

	for i, f := range []struct {
		outputPath string
		expected   string
	}{
		{
			outputPath: "../" + sampleDirectory,
			expected:   "/" + filepath.Join(sampleDirectory, asOutputFile(sampleDirectory)),
		},
		{
			outputPath: "/" + sampleDirectory,
			expected:   "/" + filepath.Join(sampleDirectory, asOutputFile(sampleDirectory)),
		},
		{
			outputPath: ".",
			expected:   "/" + asOutputFile(sampleDirectory),
		},
		{
			outputPath: "",
			expected:   "/" + asOutputFile(sampleDirectory),
		},
	} {
		f := f // pin
		t.Run(fmt.Sprintf("Index-%d", i), func(t *testing.T) {
			t.Parallel()
			a := &application{
				fs:         aferox.NewAferox("/", fs),
				outputPath: f.outputPath,
				overwrite:  true,
			}

			err := a.args(nil, []string{"../" + sampleDirectory})
			if assert.NoError(t, err) {
				if assert.Equal(t, f.expected, a.outputPath) {
					assert.NotContains(t, a.mediaFiles, a.outputPath)
				}
			}
		})
	}
}

func TestOutputFileFromRootDirectory1(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	expectedMediaFiles := []string{
		"/" + validFileName1,
		"/" + validFileName2,
	}

	for _, f := range expectedMediaFiles {
		afero.WriteFile(fs, f, defaultFileContent, 0644)
	}

	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, asOutputFile(sampleDirectory)), defaultFileContent, 0644)

	a := &application{
		fs: aferox.NewAferox("/", fs),
	}

	err := a.args(nil, []string{"/"})
	if assert.NoError(t, err) {
		if assert.Equal(t, "/"+asOutputFile(rootDirectoryName), a.outputPath) {
			if assert.NotContains(t, a.mediaFiles, a.outputPath) {
				assert.Equal(t, expectedMediaFiles, a.mediaFiles)
			}
		}
	}
}
