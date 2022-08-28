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
	supportedLanguage = "en-US"

	sampleDirectory = "sampleDirectory"

	validFileName1 = "validSampleFile1.mp3"
	validFileName2 = "validSampleFile2.mp3"
	validFileName3 = "validSampleFile3.mp3"

	invalidFileName1 = "invalidSampleFile1.mp33"
	invalidFileName2 = "invalidSampleFile2.mp33"

	validInterlaceFile1 = "interlace.mp3"
	validInterlaceFile2 = "_interlace.mp3"
)

var emptyFileContent = []byte{}

func filepathJoin(path string, files ...string) []string {
	mediaFiles := make([]string, len(files))
	for i, f := range files {
		mediaFiles[i] = filepath.Join(path, f)
	}

	return mediaFiles
}

func makeEmptyFiles(fs afero.Fs, path string, files ...string) []string {
	for _, f := range files {
		afero.WriteFile(fs, filepath.Join(path, f), emptyFileContent, 0o644)
	}

	return filepathJoin(path, files...)
}

func withTwoValidFiles(fs afero.Fs, path string) []string {
	return makeEmptyFiles(fs, path, validFileName1, validFileName2)
}

func withThreeValidFiles(fs afero.Fs, path string) []string {
	return makeEmptyFiles(fs, path, validFileName1, validFileName2, validFileName3)
}

type testTagResolver struct{}

func (t *testTagResolver) DescriptionFor(string) (string, error) {
	return "", nil
}

func newDefaultApplication(fs aferox.Aferox) *application {
	return &application{
		fs:            fs,
		languageStr:   supportedLanguage,
		statusPrinter: &discardingPrinter{},
		tagResolver:   &testTagResolver{},
	}
}

func newTestFilesystem() (string, afero.Fs) {
	fs := afero.NewMemMapFs()

	root, err := filepath.Abs(afero.FilePathSeparator)
	if err != nil {
		panic("can't get root")
	}

	fs.MkdirAll(root, 0o755)

	return root, fs
}

func TestDirectoryWithNoFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithNonExistingDirectory(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{sampleDirectory})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDirectoryWithNonExistingFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1)
	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDirectoryWithNoValidFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, invalidFileName1, invalidFileName2)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestDirectoryWithOneFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

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

			root, fs := newTestFilesystem()

			mediaFiles := makeEmptyFiles(fs, root, f...)

			a := newDefaultApplication(aferox.NewAferox(root, fs))
			a.overwrite = true

			err := a.args(nil, []string{"."})
			if assert.NoError(t, err) {
				assert.Equal(t, mediaFiles, a.mediaFiles)
			}
		})
	}
}

func TestNoParametersDefaultsToDirectory(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFiles := withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestDirectoryWithTwoFilesAndExplicitlyUsedMagicInterlaceFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1, validFileName2, validInterlaceFile1, validInterlaceFile2)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	files := []string{validFileName1, "interlace.mp3", validFileName2, "interlace.mp3", "_interlace.mp3"}
	mediaFiles := filepathJoin(root, files...)

	err := a.args(nil, files)
	if assert.NoError(t, err) {
		assert.Equal(t, len(files), len(a.mediaFiles))
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestSubDirectoryWithTwoFiles(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	fs.MkdirAll(filepath.Join(root, sampleDirectory), 0o755)
	_ = withTwoValidFiles(fs, filepath.Join(root, sampleDirectory))

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{"."})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestOneFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrAtLeastTwo)
}

func TestOneInvalidFile(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, invalidFileName1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{invalidFileName1})
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestTwoFiles(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFiles := withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestFileNotFound(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{validFileName1})
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestOneFileAndDirectory(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = makeEmptyFiles(fs, root, validFileName1)
	fs.MkdirAll(filepath.Join(root, validFileName2), 0o755)

	a := newDefaultApplication(aferox.NewAferox(root, fs))

	err := a.args(nil, []string{validFileName1, validFileName2})
	assert.Error(t, err, ErrAtLeastTwo)
}

func TestFilesAndDirectoryNoDuplicates(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFiles := withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{".", validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFiles, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryDirectoryFirst(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withThreeValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{".", validFileName1, validFileName2})
	if assert.NoError(t, err) {
		assert.Equal(t, filepathJoin(root, validFileName3, validFileName1, validFileName2), a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryLast(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withThreeValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{validFileName2, validFileName1, "."})
	if assert.NoError(t, err) {
		assert.Equal(t, filepathJoin(root, validFileName2, validFileName1, validFileName3), a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryFirst(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFilesOrdered := makeEmptyFiles(fs, root, validFileName3, validFileName2, validFileName1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{".", validFileName2, validFileName1})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFilesOrdered, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueWithExtraFromDirectoryFirstAndLast(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	mediaFilesOrdered := makeEmptyFiles(fs, root, validFileName3, validFileName2, validFileName1)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{".", validFileName2, validFileName1, "."})
	if assert.NoError(t, err) {
		assert.Equal(t, mediaFilesOrdered, a.mediaFiles)
	}
}

func TestFilesAndDirectoryUniqueButKeepDuplicatesFromArg(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	_ = withTwoValidFiles(fs, root)

	a := newDefaultApplication(aferox.NewAferox(root, fs))
	a.overwrite = true

	err := a.args(nil, []string{
		".",
		validFileName1,
		validFileName1,
		validFileName2,
		validFileName2,
	})
	if assert.NoError(t, err) {
		assert.Equal(t, filepathJoin(root, validFileName1, validFileName1, validFileName2, validFileName2), a.mediaFiles)
	}
}
