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
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputFile: validOutputFile,
	}

	err := a.args(nil, []string{"."})
	assert.NoError(t, err)
}

func TestOutputFileExistingNoOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validOutputFile, []byte("out"), 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputFile: validOutputFile,
	}

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrOutputFileExists)
}

func TestOutputFileExistingOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)
	afero.WriteFile(fs, "/"+validOutputFile, []byte("out"), 0644)

	a := &application{
		fs:         aferox.NewAferox("/", fs),
		outputFile: validOutputFile,
		overwrite:  true,
	}

	err := a.args(nil, []string{"."})
	assert.NoError(t, err)
}

func TestAsOutputFile(t *testing.T) {
	const (
		file      = "file"
		extension = ".mp3"
		fullName  = file + extension
	)

	assert.Equal(t, fullName, asOutputFile(file))
	assert.Equal(t, fullName, asOutputFile(fullName))
}

func TestOutputFileFromSampleDirectory1(t *testing.T) {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName1), []byte("1"), 0644)
	afero.WriteFile(fs, "/"+filepath.Join(sampleDirectory, validFileName2), []byte("2"), 0644)

	for i, fixture := range []struct {
		outputFile string
		expected   string
	}{
		{
			outputFile: "../" + sampleDirectory,
			expected:   "/" + filepath.Join(sampleDirectory, sampleDirectory+".mp3"),
		},
		{
			outputFile: "/" + sampleDirectory,
			expected:   "/" + filepath.Join(sampleDirectory, sampleDirectory+".mp3"),
		},
		{
			outputFile: ".",
			expected:   "/" + sampleDirectory + ".mp3",
		},
		{
			outputFile: "",
			expected:   "/" + sampleDirectory + ".mp3",
		},
	} {
		t.Run(fmt.Sprintf("Index-%d", i), func(t *testing.T) {
			a := &application{
				fs:         aferox.NewAferox("/", fs),
				outputFile: fixture.outputFile,
			}

			err := a.args(nil, []string{"../" + sampleDirectory})
			if assert.NoError(t, err) {
				assert.Equal(t, fixture.expected, a.outputFile)
			}
		})
	}
}

func TestOutputFileFromRootDirectory1(t *testing.T) {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/"+sampleDirectory, 0755)
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	a := &application{
		fs:        aferox.NewAferox("/", fs),
		overwrite: true,
	}

	err := a.args(nil, []string{"/"})
	if assert.NoError(t, err) {
		assert.Equal(t, "/root.mp3", a.outputFile)
	}
}
