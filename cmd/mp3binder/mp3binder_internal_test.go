package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crra/mp3binder/mp3binder"
	"github.com/stretchr/testify/assert"
)

var (
	arbitraryName = "_test_"

	fileInfoDirectory = &fileMock{name: arbitraryName, isDir: true}
	fileInfoFile      = &fileMock{name: arbitraryName, isDir: false}

	arbitraryMediaFile = "test" + extOfMp3
	fileInfoMediaFile  = &fileMock{name: arbitraryMediaFile, isDir: false}

	arbitraryMediaFile1 = "test1" + extOfMp3
	fileInfoMediaFile1  = &fileMock{name: arbitraryMediaFile1, isDir: false}

	arbitraryInterlaceFile     = "interlace" + extOfMp3
	fileInfoInterlaceFile      = &fileMock{name: arbitraryInterlaceFile, isDir: false}
	fileInfoMagicInterlaceFile = &fileMock{name: magicInterlaceFileName, isDir: false}

	// TODO: when on windows check: https://stackoverflow.com/questions/41254463/how-to-get-system-root-directory-for-windows-in-google-golang
	arbitraryMediaFileAbs  = "/root/testfolder/test" + extOfMp3
	arbitraryMediaFile1Abs = "/root/testfolder/test1" + extOfMp3
	arbitraryOutputFile    = "/root/testfolder/testfolder" + extOfMp3
	arbitraryFileList      = []string{
		"foo" + extOfMp3,
		"bar" + extOfMp3,
		"baz" + extOfMp3,
	}
	arbitraryPath               = filepath.Join("foo", "bar", "baz")
	arbitraryOutputFileFromPath = filepath.Join("foo", "bar", "baz", "baz.mp3")
	validCoverFile              = "cover.jpg"
)

func checkError(t *testing.T, err, expectedError error) {
	if err != nil {
		if expectedError != nil {
			assert.Error(t, err)
			assert.True(t,
				strings.Contains(err.Error(), expectedError.Error()),
				"error message not equal",
				"expected:", expectedError.Error(),
				"received:", err.Error(),
			)
		} else {
			assert.Fail(t, "Got error, but non was expected. Message: ", err.Error())
		}
	}
}

// The errors message serve a dual use:
// 1. they are returned as a non-nil error
// 2. the message is used as a substring of the application error message
//    that is different from the framework error. E.g. the error may contain
//    the file or directory in question
var (
	errArbitrary                 = fmt.Errorf("a non-nil error")
	errNotADirectory             = fmt.Errorf("is not a directory")
	errNonExisting               = fmt.Errorf("does not exist")
	errFailedAbs                 = fmt.Errorf("can not get absolute path")
	errFailedRead                = fmt.Errorf("can not read files from directory")
	errNoMediaFile               = fmt.Errorf("is not a media file")
	errNoMediaFiles              = fmt.Errorf("no media files")
	errOnlyOneMediaFile          = fmt.Errorf("only one media file")
	errAlreadyExists             = fmt.Errorf("file already exists")
	errCoverNotSupported         = fmt.Errorf("is not supported")
	errMetadataIndexInvalid      = fmt.Errorf("copying metadata is invalid")
	errCanNotDetermineOutputFile = fmt.Errorf("can not determine output file from input")
	errNoInput                   = fmt.Errorf("no input specified")
)

type fileSystemMock struct {
	stat    fileStatFn
	readDir dirReaderFn
	abs     absFn
}

func newFileSystemMock(stat fileStatFn, readDir dirReaderFn, abs absFn) *fileSystemMock {
	return &fileSystemMock{
		stat:    stat,
		readDir: readDir,
		abs:     abs,
	}
}

func (fsm *fileSystemMock) Stat(name string) (os.FileInfo, error) { return fsm.stat(name) }
func (fsm *fileSystemMock) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fsm.readDir(dirname)
}
func (fsm *fileSystemMock) Abs(path string) (string, error) { return fsm.abs(path) }

func newAbsMock(err error) absFn {
	return func(path string) (string, error) {
		return path, err
	}
}

var (
	noErrorAbs = newAbsMock(nil)
	failingAbs = newAbsMock(errFailedAbs)
)

func newSimpleFileStatMock(info os.FileInfo, err error) fileStatFn {
	return func(ignore string) (os.FileInfo, error) {
		return info, err
	}
}

type fileInfoErrMock struct {
	info os.FileInfo
	err  error
}

func newMapFileStatMock(mocks map[string]fileInfoErrMock) fileStatFn {
	return func(path string) (os.FileInfo, error) {
		mock, ok := mocks[path]
		if !ok {
			panic(fmt.Sprintf("mock for %s not defined", path))
		}

		return mock.info, mock.err
	}
}

type fileMock struct {
	os.FileInfo
	name  string
	isDir bool
}

func (m *fileMock) Name() string {
	return m.name
}

func (m *fileMock) IsDir() bool {
	return m.isDir
}

func newDirectoryReaderMock(infos []os.FileInfo, err error) dirReaderFn {
	return func(directory string) ([]os.FileInfo, error) {
		return infos, err
	}
}

var (
	fileInfoFilterAcceptAll        = func(info os.FileInfo) bool { return true }
	fileInfoFilterRejectAll        = func(info os.FileInfo) bool { return false }
	fileInfoFilterAcceptFiles      = func(info os.FileInfo) bool { return !info.IsDir() }
	fileInfoFilterAcceptMediaFiles = func(file os.FileInfo) bool {
		return !file.IsDir() && strings.EqualFold(extOfMp3, filepath.Ext(file.Name()))
	}
)

func newFsDirReaderMock(info []os.FileInfo, err error) dirReaderFn {
	return func(ignore string) ([]os.FileInfo, error) {
		return info, err
	}
}

const (
	nonRelevantDirectoryReaderDirectories = 2
	nonRelevantDirectoryReaderFiles       = 3
	nonRelevantDirectoryReaderTotal       = nonRelevantDirectoryReaderDirectories + nonRelevantDirectoryReaderFiles
)

var nonRelevantDirectoryReader = newFsDirReaderMock([]os.FileInfo{
	&fileMock{name: "file1", isDir: false},
	&fileMock{name: "file2", isDir: false},
	&fileMock{name: "file3", isDir: false},
	&fileMock{name: "dir1", isDir: true},
	&fileMock{name: "dir2", isDir: true},
}, nil)

func TestUserInput(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title         string
		args          []string
		expected      *userInput
		expectedError *error
	}{
		{
			title: "empty input",
			args:  []string{"test"},
			expected: &userInput{
				printInformation:     true,
				discoverMagicFiles:   true,
				commandLineArguments: []string{},
			},
		},
		{
			title:         "raise help error",
			args:          []string{arbitraryName, "-help"},
			expectedError: &flag.ErrHelp,
		},
		{
			title: "suppress output",
			args:  []string{arbitraryName, "-q"},
			expected: &userInput{
				printInformation:     false,
				discoverMagicFiles:   true,
				commandLineArguments: []string{},
			},
		},
		{
			title: "no magic",
			args:  []string{arbitraryName, "-noauto"},
			expected: &userInput{
				printInformation:     true,
				discoverMagicFiles:   false,
				commandLineArguments: []string{},
			},
		},
		{
			title: "force overwrite",
			args:  []string{arbitraryName, "-f"},
			expected: &userInput{
				printInformation:     true,
				discoverMagicFiles:   true,
				forceOverwrite:       true,
				commandLineArguments: []string{},
			},
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			input, err := newUserInput(f.args, &b, flag.ContinueOnError)

			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.expected, input)
		})
	}
}
func TestFileListReader(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title         string
		files         []string
		fs            fileSystemAbstraction
		filter        fileInfoFilterFn
		expectedError *error
		numberOfFiles int
	}{
		{
			title: "empty input",
		},
		{
			title:         "failing abs",
			files:         []string{"test"},
			fs:            newFileSystemMock(nil, nil, failingAbs),
			expectedError: &errFailedAbs,
		},
		{
			title: "non-existing",
			files: []string{"test"},
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoFile, errNonExisting),
				nil,
				noErrorAbs),
			expectedError: &errNonExisting,
		},
		{
			title:         "invalid",
			files:         []string{"test"},
			filter:        fileInfoFilterRejectAll,
			fs:            newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, noErrorAbs),
			expectedError: &errNoMediaFile,
		},
		{
			title:         "success: valid",
			files:         []string{"test", "test", "test"},
			filter:        fileInfoFilterAcceptFiles,
			fs:            newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, noErrorAbs),
			numberOfFiles: 3,
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			files, err := fileListReader(f.fs, f.files, f.filter)

			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.numberOfFiles, len(files))
		})
	}
}

func TestReadDirectory(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title         string
		fs            fileSystemAbstraction
		filter        fileInfoFilterFn
		first         bool
		expectedError *error
		numberOfFiles int
	}{
		{
			title:  "empty directory",
			filter: fileInfoFilterAcceptAll,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				newFsDirReaderMock([]os.FileInfo{}, nil),
				noErrorAbs),
		},
		{
			title: "file instead of directory",
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoFile, nil),
				newFsDirReaderMock([]os.FileInfo{}, nil),
				noErrorAbs),
			expectedError: &errNotADirectory,
		},
		{
			title: "non-existing",
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoFile, errNonExisting),
				newFsDirReaderMock([]os.FileInfo{}, nil),
				noErrorAbs),
			expectedError: &errNonExisting,
		},
		{
			title: "failed abs",
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				newFsDirReaderMock([]os.FileInfo{}, nil),
				failingAbs),
			expectedError: &errFailedAbs,
		},
		{
			title:  "error opening directory",
			filter: fileInfoFilterAcceptAll,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				newFsDirReaderMock([]os.FileInfo{}, errFailedRead),
				noErrorAbs),
			expectedError: &errFailedRead,
		},
		{
			title:  "success: accepting all filter",
			filter: fileInfoFilterAcceptAll,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				nonRelevantDirectoryReader,
				noErrorAbs),
			numberOfFiles: nonRelevantDirectoryReaderTotal,
		},
		{
			title:  "success: rejecting all filter",
			filter: fileInfoFilterRejectAll,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				nonRelevantDirectoryReader,
				noErrorAbs),
		},
		{
			title:  "success: accepting first",
			filter: fileInfoFilterAcceptAll,
			first:  true,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				nonRelevantDirectoryReader,
				noErrorAbs),
			numberOfFiles: 1,
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			files, err := directoryReader(f.fs, arbitraryPath, f.filter, f.first)

			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.numberOfFiles, len(files))
		})
	}
}

func TestOutputFilename(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title            string
		fs               fileSystemAbstraction
		outputFileName   *string
		forceOverwrite   bool
		inputDirectory   *string
		files            []string
		expectedError    *error
		expectedFileName string
	}{
		{
			title:          "failed abs",
			outputFileName: &arbitraryName,
			fs:             newFileSystemMock(nil, nil, failingAbs),
			expectedError:  &errFailedAbs,
		},
		{
			title:            "existing defined file name",
			outputFileName:   &arbitraryName,
			expectedFileName: "",
			fs:               newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, noErrorAbs),
			expectedError:    &errAlreadyExists,
		},
		{
			title:         "can not derive",
			expectedError: &errCanNotDetermineOutputFile,
		},
		{
			title:            "success: defined non existing file",
			outputFileName:   &arbitraryName,
			expectedFileName: arbitraryName,
			fs:               newFileSystemMock(newSimpleFileStatMock(fileInfoFile, errNonExisting), nil, noErrorAbs),
		},
		{
			title:            "success: overwrite existing defined file name",
			outputFileName:   &arbitraryName,
			expectedFileName: arbitraryName,
			forceOverwrite:   true,
			fs:               newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, noErrorAbs),
		},
		{
			title:            "success: determine filename",
			inputDirectory:   &arbitraryPath,
			expectedFileName: filepath.Join(arbitraryPath, filepath.Base(arbitraryPath)+extOfMp3),
			fs:               newFileSystemMock(newSimpleFileStatMock(fileInfoFile, errNonExisting), nil, noErrorAbs),
		},
		{
			title:            "success: determine filename from command line files",
			expectedFileName: arbitraryOutputFile,
			fs:               newFileSystemMock(newSimpleFileStatMock(fileInfoFile, errNonExisting), nil, noErrorAbs),
			files:            []string{arbitraryMediaFileAbs},
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			outputFileName, err := getOutputFileName(f.fs, &b, f.outputFileName, f.forceOverwrite, f.inputDirectory, f.files)

			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.expectedFileName, outputFileName)
		})
	}
}

func TestEnsureProcessableFiles(t *testing.T) {
	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		checkError(t, ensureProcessableFiles(0), errNoMediaFiles)
	})
	t.Run("one", func(t *testing.T) {
		t.Parallel()

		checkError(t, ensureProcessableFiles(1), errOnlyOneMediaFile)
	})
	t.Run("many", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, ensureProcessableFiles(42))
	})
}

func TestCoverFile(t *testing.T) {
	t.Parallel()

	validCoverFileAbs := filepath.Join(arbitraryName, validCoverFile)

	testTable := []struct {
		title              string
		fs                 fileSystemAbstraction
		coverFileName      *string
		discoverMagicFiles bool
		inputDirectory     *string
		expectedError      *error
		expectedCoverFile  *string
	}{
		{
			title: "none",
		},
		{
			title:         "non-existing",
			coverFileName: &validCoverFile,
			fs:            newFileSystemMock(newSimpleFileStatMock(fileInfoFile, errNonExisting), nil, nil),
			expectedError: &errNonExisting,
		},
		{
			title:         "failed abs",
			coverFileName: &validCoverFile,
			fs:            newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, failingAbs),
			expectedError: &errFailedAbs,
		},
		{
			title:         "existing invalid",
			coverFileName: &arbitraryName,
			fs:            newFileSystemMock(newSimpleFileStatMock(fileInfoFile, nil), nil, noErrorAbs),
			expectedError: &errCoverNotSupported,
		},
		{
			title:         "success: existing valid",
			coverFileName: &validCoverFile,
			fs: newFileSystemMock(
				newSimpleFileStatMock(&fileMock{name: validCoverFile, isDir: false}, nil), nil, noErrorAbs),
			expectedCoverFile: &validCoverFile,
		},
		{
			title:              "discover failed dir reader",
			discoverMagicFiles: true,
			inputDirectory:     &arbitraryName,
			fs:                 newFileSystemMock(nil, nil, failingAbs),
			expectedError:      &errFailedAbs,
		},
		{
			title:              "success: discover",
			discoverMagicFiles: true,
			inputDirectory:     &arbitraryName,
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoDirectory, nil),
				newDirectoryReaderMock([]os.FileInfo{
					fileInfoFile,
					fileInfoDirectory,
					&fileMock{name: arbitraryName, isDir: false},
					&fileMock{name: validCoverFile, isDir: false},
				}, nil),
				noErrorAbs),
			expectedCoverFile: &validCoverFileAbs,
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer

			coverFile, err := getCoverFileName(f.fs, &b, f.coverFileName, f.discoverMagicFiles, f.inputDirectory)

			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)
			assert.Equal(t, f.expectedCoverFile, coverFile)
		})
	}
}

func TestInterlaceFile(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title                     string
		fs                        fileSystemAbstraction
		filter                    fileInfoFilterFn
		interlaceFileName         *string
		discoverMagicFiles        bool
		files                     []string
		expectedError             *error
		expectedInterlaceFileName *string
	}{
		{
			title: "no interlace file",
		},
		{
			title:             "failing abs",
			interlaceFileName: &arbitraryInterlaceFile,
			fs:                newFileSystemMock(nil, nil, failingAbs),
			expectedError:     &errFailedAbs,
		},
		{
			title:             "non-existing interlace file",
			interlaceFileName: &arbitraryInterlaceFile,
			fs:                newFileSystemMock(newSimpleFileStatMock(fileInfoFile, errNonExisting), nil, noErrorAbs),
			expectedError:     &errNonExisting,
		},
		{
			title:                     "success: existing interlace file",
			interlaceFileName:         &arbitraryInterlaceFile,
			filter:                    fileInfoFilterAcceptMediaFiles,
			fs:                        newFileSystemMock(newSimpleFileStatMock(fileInfoInterlaceFile, nil), nil, noErrorAbs),
			expectedInterlaceFileName: &arbitraryInterlaceFile,
		},
		{
			title:              "failing magic file discovery",
			discoverMagicFiles: true,
			files:              arbitraryFileList,
		},
		{
			title:                     "success: discover magic file",
			discoverMagicFiles:        true,
			filter:                    fileInfoFilterAcceptMediaFiles,
			files:                     append(arbitraryFileList, magicInterlaceFileName),
			fs:                        newFileSystemMock(newSimpleFileStatMock(fileInfoMagicInterlaceFile, nil), nil, noErrorAbs),
			expectedInterlaceFileName: stringToPtr(magicInterlaceFileName),
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer

			interlaceFileName, err := getInterlaceFileName(f.fs, &b, f.interlaceFileName, f.filter, f.discoverMagicFiles, f.files)
			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.expectedInterlaceFileName, interlaceFileName)
		})
	}
}

func TestRemoveElementsFromStringList(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title            string
		list             []string
		elementsToRemove []string
		expectedList     []string
	}{
		{
			title:            "a,b,c",
			list:             []string{"a", "b", "c"},
			elementsToRemove: []string{"a"},
			expectedList:     []string{"b", "c"},
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, f.expectedList, removeElementsFromStringList(f.list, f.elementsToRemove))
		})
	}
}

func TestGetElementByIndex(t *testing.T) {
	t.Parallel()

	firstElement := "first"
	lastElement := "last"

	testTable := []struct {
		title           string
		list            []string
		index           int
		expectedError   *error
		expectedElement *string
	}{
		{
			title:         "out of bounds neg",
			list:          []string{},
			index:         -1,
			expectedError: &errMetadataIndexInvalid,
		},
		{
			title:         "out of bounds pos",
			list:          []string{},
			index:         1,
			expectedError: &errMetadataIndexInvalid,
		},
		{
			title:         "zero empty",
			list:          []string{},
			index:         0,
			expectedError: &errMetadataIndexInvalid,
		},
		{
			title:           "success: zero first",
			list:            []string{firstElement},
			index:           0,
			expectedElement: &firstElement,
		},
		{
			title:           "success: one first",
			list:            []string{firstElement},
			index:           1,
			expectedElement: &firstElement,
		},
		{
			title:           "success: last",
			list:            []string{firstElement, lastElement},
			index:           2,
			expectedElement: &lastElement,
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			element, err := getElementByIndex(&b, f.list, f.index)
			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.expectedElement, element)
		})
	}
}

func TestInterlaceFiles(t *testing.T) {
	t.Parallel()

	interlace := "interlace"

	testTable := []struct {
		title        string
		list         []string
		interlace    string
		expectedList []string
	}{
		{
			title:        "empty list",
			list:         []string{},
			interlace:    interlace,
			expectedList: []string{},
		},
		{
			title:        "one element",
			list:         []string{"one"},
			interlace:    interlace,
			expectedList: []string{"one"},
		},
		{
			title:        "success: two elements",
			list:         []string{"one", "two"},
			interlace:    interlace,
			expectedList: []string{"one", interlace, "two"},
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, f.expectedList, addInterlaceToStringList(f.list, f.interlace))
		})
	}
}

func TestKvStringToTagMap(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title           string
		kvpString       string
		expectedMap     map[string]string
		expectedWarning string
	}{
		{
			title:       "empty string",
			expectedMap: make(map[string]string),
		},
		{
			title:           "malformed",
			kvpString:       "foo=bar=baz",
			expectedMap:     make(map[string]string),
			expectedWarning: "is not in the form",
		},
		{
			title:           "not well known",
			kvpString:       "foo=bar",
			expectedMap:     make(map[string]string),
			expectedWarning: "is not a well-known tag",
		},
		{
			title:     "success: well-known",
			kvpString: "TIT2=title",
			expectedMap: map[string]string{
				"TIT2": "title",
			},
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			assert.Equal(t, f.expectedMap, kvStringToTagMap(&b, f.kvpString))
			assert.True(t, strings.Contains(b.String(), f.expectedWarning))
		})
	}
}

func TestMimeTypes(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		file string
		mime string
	}{
		{file: "test.mp3", mime: "audio/mpeg"},
		{file: "test.png", mime: "image/png"},
		{file: "test.jpg", mime: "image/jpeg"},
		{file: "test.jpeg", mime: "image/jpeg"},
		{file: arbitraryName, mime: "application/octet-stream"},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.mime, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, f.mime, getMimeFromFileName(f.file))
		})
	}
}

func TestJobCreation(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		title         string
		fs            fileSystemAbstraction
		input         *userInput
		expectedError *error
		expectedJob   *job
	}{
		{
			title:         "no input",
			expectedError: &errNoInput,
		},
		{
			title: "no media file (via command line)",
			input: &userInput{
				commandLineArguments: []string{arbitraryName},
			},
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoFile, nil),
				nil,
				noErrorAbs,
			),
			expectedError: &errNoMediaFile,
		},
		{
			title: "only one file (via command line)",
			input: &userInput{
				commandLineArguments: []string{arbitraryMediaFile},
			},
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoMediaFile, nil),
				nil,
				noErrorAbs,
			),
			expectedError: &errOnlyOneMediaFile,
		},
		{
			title: "interlace file same as two input files, expect no media files",
			input: &userInput{
				commandLineArguments: []string{
					arbitraryMediaFileAbs,
					arbitraryMediaFileAbs,
				},
				interlaceFileName: &arbitraryMediaFileAbs,
			},
			fs: newFileSystemMock(
				newSimpleFileStatMock(fileInfoFile, nil),
				nil,
				noErrorAbs,
			),
			expectedError: &errNoMediaFile,
		},
		{
			title: "interlace file part of input, will result in only one media file",
			input: &userInput{
				commandLineArguments: []string{
					arbitraryMediaFileAbs,
					arbitraryInterlaceFile,
				},
				interlaceFileName: &arbitraryInterlaceFile,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:  {info: fileInfoMediaFile},
					arbitraryInterlaceFile: {info: fileInfoInterlaceFile},
					arbitraryOutputFile:    {err: errArbitrary},
				}),
				nil,
				noErrorAbs,
			),
			expectedError: &errOnlyOneMediaFile,
		},
		{
			title: "invalid input directory",
			input: &userInput{
				inputDirectory: &arbitraryPath,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryPath: {err: errArbitrary},
				}),

				nil,
				noErrorAbs,
			),
			expectedError: &errNonExisting,
		},
		{
			title: "invalid input directory",
			input: &userInput{
				inputDirectory: &arbitraryPath,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryPath: {info: fileInfoDirectory},
				}),

				nil,
				failingAbs,
			),
			expectedError: &errFailedAbs,
		},
		{
			title: "existing output file",
			input: &userInput{
				inputDirectory: &arbitraryPath,
				commandLineArguments: []string{
					arbitraryMediaFile1Abs,
					arbitraryMediaFileAbs,
				},
				outputFileName: &arbitraryOutputFileFromPath,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:       {info: fileInfoMediaFile},
					arbitraryMediaFile1Abs:      {info: fileInfoMediaFile1},
					arbitraryPath:               {info: fileInfoDirectory},
					arbitraryOutputFileFromPath: {},
				}),
				nonRelevantDirectoryReader,
				noErrorAbs,
			),
			expectedError: &errAlreadyExists,
		},
		{
			title: "invalid cover file",
			input: &userInput{
				inputDirectory: &arbitraryPath,
				commandLineArguments: []string{
					arbitraryMediaFile1Abs,
					arbitraryMediaFileAbs,
				},
				outputFileName: &arbitraryOutputFileFromPath,
				coverFileName:  &arbitraryName,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:       {info: fileInfoMediaFile},
					arbitraryMediaFile1Abs:      {info: fileInfoMediaFile1},
					arbitraryPath:               {info: fileInfoDirectory},
					arbitraryOutputFileFromPath: {err: errArbitrary},
					arbitraryName:               {info: fileInfoFile},
				}),
				nonRelevantDirectoryReader,
				noErrorAbs,
			),
			expectedError: &errCoverNotSupported,
		},
		{
			title: "invalid interlace file",
			input: &userInput{
				inputDirectory: &arbitraryPath,
				commandLineArguments: []string{
					arbitraryMediaFile1Abs,
					arbitraryMediaFileAbs,
				},
				outputFileName:    &arbitraryOutputFileFromPath,
				interlaceFileName: &arbitraryName,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:       {info: fileInfoMediaFile},
					arbitraryMediaFile1Abs:      {info: fileInfoMediaFile1},
					arbitraryPath:               {info: fileInfoDirectory},
					arbitraryOutputFileFromPath: {err: errArbitrary},
					arbitraryName:               {info: fileInfoFile},
				}),
				nonRelevantDirectoryReader,
				noErrorAbs,
			),
			expectedError: &errNoMediaFile,
		},
		{
			title: "invalid index for template file",
			input: &userInput{
				inputDirectory: &arbitraryPath,
				commandLineArguments: []string{
					arbitraryMediaFile1Abs,
					arbitraryMediaFileAbs,
				},
				copyMetadataFromFileIndex: IntToPtr(42),
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:       {info: fileInfoMediaFile},
					arbitraryMediaFile1Abs:      {info: fileInfoMediaFile1},
					arbitraryPath:               {info: fileInfoDirectory},
					arbitraryOutputFileFromPath: {err: errArbitrary},
				}),
				nonRelevantDirectoryReader,
				noErrorAbs,
			),
			expectedError: &errMetadataIndexInvalid,
		},
		{
			title: "success: with files, with interlace, with cover, with tags, with template file",
			input: &userInput{
				inputDirectory: &arbitraryPath,
				commandLineArguments: []string{
					arbitraryMediaFile1Abs,
					arbitraryMediaFileAbs,
				},
				interlaceFileName:         &arbitraryInterlaceFile,
				coverFileName:             &validCoverFile,
				copyMetadataFromFileIndex: IntToPtr(1),
				tags:                      stringToPtr(fmt.Sprintf("%s=42", mp3binder.TagTrack)),
			},
			expectedJob: &job{
				files: []string{
					arbitraryMediaFile1Abs,
					arbitraryInterlaceFile,
					arbitraryMediaFileAbs,
				},
				outputFileName:  arbitraryOutputFileFromPath,
				coverFileName:   &validCoverFile,
				tags:            map[string]string{mp3binder.TagTrack: "42"},
				tagTemplateFile: &arbitraryMediaFile1Abs,
			},
			fs: newFileSystemMock(
				newMapFileStatMock(map[string]fileInfoErrMock{
					arbitraryMediaFileAbs:       {info: fileInfoMediaFile},
					arbitraryMediaFile1Abs:      {info: fileInfoMediaFile1},
					arbitraryOutputFileFromPath: {err: errArbitrary},
					validCoverFile:              {},
					arbitraryInterlaceFile:      {info: fileInfoInterlaceFile},
					arbitraryPath:               {info: fileInfoDirectory},
				}),
				nonRelevantDirectoryReader,
				noErrorAbs,
			),
		},
	}

	for _, fixture := range testTable {
		f := fixture
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()

			if f.title == "interlace file part of input, will result in no media files" {
				fmt.Print("Here")
			}

			var b bytes.Buffer
			job, err := newJobFromInput(f.fs, f.input, &b)
			var expected error
			if f.expectedError != nil {
				expected = *f.expectedError
			}

			checkError(t, err, expected)

			assert.Equal(t, f.expectedJob, job)
		})
	}
}
