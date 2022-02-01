package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carolynvs/aferox"
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
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.outputPath = filepath.Join(root, validOutputFile)

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		// not changed
		assert.Equal(t, filepath.Join(root, validOutputFile), a.outputPath)
	}
}

func TestOutputFileMissingExtension(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.outputPath = fileNameWithoutExtension(validOutputFile)

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, filepath.Join(root, validOutputFile), a.outputPath)
	}
}

func TestOutputFileExistingNoOverwrite(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	_ = makeFiles(fs, root, validOutputFile)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.outputPath = validOutputFile

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrOutputFileExists)
}

func TestOutputFileExistingOverwrite(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)
	outputFiles := makeFiles(fs, root, validOutputFile)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.outputPath = outputFiles[0]
	a.overwrite = true

	err := a.args(nil, []string{"."})
	if assert.NoError(t, err) {
		assert.Equal(t, outputFiles[0], a.outputPath)
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
	root, fs := newTestFilesystem()
	samplePath := filepath.Join(root, sampleDirectory)
	fs.MkdirAll(samplePath, 0755)
	_ = withTwoValidFiles(fs, samplePath)

	for i, f := range []struct {
		outputPath string
		expected   string
	}{
		{
			outputPath: filepath.Join("..", sampleDirectory),
			expected:   filepath.Join(samplePath, asOutputFile(sampleDirectory)),
		},
		{
			outputPath: samplePath,
			expected:   filepath.Join(samplePath, asOutputFile(sampleDirectory)),
		},
		{
			outputPath: ".",
			expected:   asOutputFile(samplePath),
		},
		{
			outputPath: "",
			expected:   filepath.Join(root, asOutputFile(sampleDirectory)),
		},
	} {
		f := f // pin
		t.Run(fmt.Sprintf("Index-%d", i), func(t *testing.T) {
			t.Parallel()
			a := newDefaultApplication(aferox.NewAferox(root, fs))
			a.outputPath = f.outputPath
			a.overwrite = true

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
	root, fs := newTestFilesystem()
	samplePath := filepath.Join(root, sampleDirectory)
	fs.MkdirAll(samplePath, 0755)
	expectedMediaFiles := withTwoValidFiles(fs, root)

	_ = makeFiles(fs, samplePath, asOutputFile(sampleDirectory))

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{string(os.PathSeparator)})
	if assert.NoError(t, err) {
		if assert.Equal(t, filepath.Join(root, asOutputFile(rootDirectoryName)), a.outputPath) {
			if assert.NotContains(t, a.mediaFiles, a.outputPath) {
				assert.Equal(t, expectedMediaFiles, a.mediaFiles)
			}
		}
	}
}
