package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2"
	"github.com/crra/mp3binder/flagext"
)

var (
	// externally set by the build system.
	version = "dev-build"
	name    = "mp3builder"
	realm   = "mp3builder"
)

const (
	extOfMp3                  = ".mp3"
	interlaceFilesScaleFactor = 2
	magicInterlaceFileName    = "_interlace.mp3"

	keyValuePairSize = 2
	pairSeparator    = ","
	valueSeparator   = "="
)

var (
	artworkCoverFiles       = make(map[string]struct{})
	supportedCoverMimeTypes = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
	}
)

func init() {
	artworkNames := []string{"cover", "folder", "artwork"}

	for _, ext := range supportedCoverMimeTypes {
		for _, name := range artworkNames {
			artworkCoverFiles[name+ext] = struct{}{}
		}
	}
}

func stringToPtr(in string) *string { return &in }
func IntToPtr(in int) *int          { return &in }

type (
	fileStatFn  func(string) (os.FileInfo, error)
	dirReaderFn func(string) ([]os.FileInfo, error)
	absFn       func(string) (string, error)
)

type fileSystemAbstraction interface {
	Stat(name string) (os.FileInfo, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Abs(path string) (string, error)
}

func newUserInput(arguments []string, output io.Writer, errorHandling flag.ErrorHandling) (*userInput, error) {
	input := &userInput{}

	fs := flag.NewFlagSet(arguments[0], errorHandling)
	fs.SetOutput(output)

	fs.BoolVar(&input.showVersion, "version", false, "show version info")
	fs.BoolVar(&input.forceOverwrite, "f", false, "overwrite an existing output file")
	quiet := fs.Bool("q", false, "suppress info and warnings")
	noAuto := fs.Bool("noauto", false, "disable auto discovering of magic files")

	fs.Var(flagext.NewStringPtrFlag(&input.outputFileName), "out",
		"output filepath. Defaults to name of the folder of the first file provided")
	fs.Var(flagext.NewStringPtrFlag(&input.inputDirectory), "dir", "directory of files to merge")

	fs.Var(flagext.NewStringPtrFlag(&input.interlaceFileName), "interlace",
		"interlace a spacer file (e.g. silence) between each input file")
	fs.Var(flagext.NewStringPtrFlag(&input.coverFileName), "cover", "use image file as artwork")
	fs.Var(flagext.NewIntPtrFlag(&input.copyMetadataFromFileIndex), "tcopy",
		"copy the ID3 metadata tag from the n-th input file, starting with 1")
	fs.Var(flagext.NewStringPtrFlag(&input.tags), "tapply",
		"apply id3v2 tags to output file.\n"+
			"Takes the format 'key1=value,key2=value'.\n"+
			"Keys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames")

	if err := fs.Parse(arguments[1:]); err != nil {
		return nil, err
	}

	input.commandLineArguments = fs.Args()
	input.discoverMagicFiles = !*noAuto
	input.printInformation = !*quiet

	return input, nil
}

type userInput struct {
	showVersion        bool
	printInformation   bool
	forceOverwrite     bool
	discoverMagicFiles bool

	commandLineArguments      []string
	inputDirectory            *string
	interlaceFileName         *string
	outputFileName            *string
	coverFileName             *string
	copyMetadataFromFileIndex *int
	tags                      *string
}

type job struct {
	files           []string
	outputFileName  string
	coverFileName   *string
	tagTemplateFile *string
	tags            map[string]string
}

func newJobFromInput(fs fileSystemAbstraction, input *userInput, infoWriter io.Writer) (*job, error) {
	var (
		err               error
		commandLineFiles  []string
		directoryFiles    []string
		files             []string
		outputFileName    string
		coverFileName     *string
		interlaceFileName *string
		tagTemplateFile   *string
		tags              map[string]string
		filter            = func(file os.FileInfo) bool {
			return !file.IsDir() && strings.EqualFold(extOfMp3, filepath.Ext(file.Name()))
		}
	)

	if input == nil {
		return nil, fmt.Errorf("no input specified")
	}

	if commandLineFiles, err = fileListReader(fs, input.commandLineArguments, filter); err != nil {
		return nil, err
	}

	if input.inputDirectory != nil {
		var inputDirectoryAbs string

		if inputDirectoryAbs, err = fs.Abs(*input.inputDirectory); err != nil {
			return nil, fmt.Errorf("can not get absolute location of file '%s', %v", *input.inputDirectory, err)
		}

		input.inputDirectory = &inputDirectoryAbs

		if directoryFiles, err = directoryReader(fs, *input.inputDirectory, filter, false); err != nil {
			return nil, err
		}
	}

	files = commandLineFiles
	files = append(files, directoryFiles...)

	if err1 := ensureProcessableFiles(len(files)); err1 != nil {
		return nil, err1
	}

	if outputFileName, err = getOutputFileName(
		fs, infoWriter, input.outputFileName, input.forceOverwrite, input.inputDirectory, files); err != nil {
		return nil, err
	}

	if coverFileName, err = getCoverFileName(fs, infoWriter, input.coverFileName,
		input.discoverMagicFiles, input.inputDirectory); err != nil {
		return nil, err
	}

	if interlaceFileName, err = getInterlaceFileName(
		fs, infoWriter, input.interlaceFileName, filter, input.discoverMagicFiles, files); err != nil {
		return nil, err
	}

	specialFiles := []string{outputFileName}
	if interlaceFileName != nil {
		specialFiles = append(specialFiles, *interlaceFileName)
	}

	files = removeElementsFromStringList(files, specialFiles)

	if err1 := ensureProcessableFiles(len(files)); err1 != nil {
		return nil, err1
	}

	if input.copyMetadataFromFileIndex != nil {
		if tagTemplateFile, err = getElementByIndex(infoWriter, files, *input.copyMetadataFromFileIndex); err != nil {
			return nil, err
		}
	}

	if interlaceFileName != nil {
		files = addInterlaceToStringList(files, *interlaceFileName)
	}

	if input.tags != nil {
		tags = kvStringToTagMap(infoWriter, *input.tags)
	}

	return &job{
		files:           files,
		outputFileName:  outputFileName,
		coverFileName:   coverFileName,
		tagTemplateFile: tagTemplateFile,
		tags:            tags,
	}, nil
}

type fileInfoFilterFn func(info os.FileInfo) bool

func fileListReader(fs fileSystemAbstraction, files []string, filter fileInfoFilterFn) ([]string, error) {
	for i, file := range files {
		if !filepath.IsAbs(file) {
			var err error

			if file, err = fs.Abs(file); err != nil {
				return []string{}, fmt.Errorf("can not get absolute path for file '%s', %v", file, err)
			}

			files[i] = file
		}

		var (
			info os.FileInfo
			err  error
		)

		if info, err = fs.Stat(file); err != nil {
			return []string{}, fmt.Errorf("given media file '%s' does not exist", file)
		}

		if !filter(info) {
			return []string{}, fmt.Errorf("given media file '%s' is not a media file", file)
		}
	}

	return files, nil
}

func directoryReader(
	fs fileSystemAbstraction,
	directory string,
	filter fileInfoFilterFn,
	first bool,
) ([]string, error) {
	var err error

	if !filepath.IsAbs(directory) {
		directory, err = fs.Abs(directory)
		if err != nil {
			return []string{}, fmt.Errorf("can not get absolute path for directory '%s', %v", directory, err)
		}
	}

	fstat, err := fs.Stat(directory)
	if err != nil {
		return []string{}, fmt.Errorf("directory does not exists '%s'", directory)
	} else if !fstat.IsDir() {
		return []string{}, fmt.Errorf("provided path for '-dir' is not a directory '%s'", directory)
	}

	dirContent, err := fs.ReadDir(directory)
	if err != nil {
		return []string{}, fmt.Errorf("can not read media files from directory, %v", err)
	}

	index := 0
	files := make([]string, len(dirContent))

	for _, info := range dirContent {
		if filter(info) {
			files[index] = filepath.Join(directory, info.Name())
			index++

			if first {
				break
			}
		}
	}

	return files[:index], nil
}

// getOutputFileName determines the file name of the for the output file
// either the name is explicitly set or the name is derived from the
// input directory or from the first file provided via the command line
func getOutputFileName(
	fs fileSystemAbstraction,
	output io.Writer,
	outputFileName *string,
	forceOverwrite bool,
	inputDirectory *string,
	files []string,
) (string, error) {
	if outputFileName == nil {
		var fileName string

		if inputDirectory != nil {
			nameOfFolder := filepath.Base(*inputDirectory)
			fileName = filepath.Join(*inputDirectory, nameOfFolder+extOfMp3)
		} else {
			if len(files) < 1 {
				return "", fmt.Errorf("can not determine output file from input, please specify with the '-out' option")
			}
			pathFromFirstFile := filepath.Dir(files[0])
			nameOfFolder := filepath.Base(pathFromFirstFile)
			fileName = filepath.Join(pathFromFirstFile, nameOfFolder+extOfMp3)
		}

		outputFileName = &fileName
	}

	absOutputFileName, err := fs.Abs(*outputFileName)
	if err != nil {
		return "", fmt.Errorf("can not get absolute location of file '%s', %v", *outputFileName, err)
	}

	_, err = fs.Stat(absOutputFileName)
	if err == nil {
		if forceOverwrite {
			fmt.Fprintf(output, "Info: overwriting file '%s'\n", *outputFileName)
		} else {
			return "", fmt.Errorf("file already exists '%s', use force (-f) to overwrite", *outputFileName)
		}
	}

	return absOutputFileName, nil
}

func ensureProcessableFiles(length int) error {
	switch length {
	case 0:
		return fmt.Errorf("no media files for processing available")
	case 1:
		return fmt.Errorf("only one media file for processing available")
	default:
		return nil
	}
}

func getCoverFileName(
	fs fileSystemAbstraction,
	output io.Writer,
	coverFileName *string,
	discoverMagicFiles bool,
	inputDirectory *string) (*string, error) {
	if coverFileName != nil {
		_, err := fs.Stat(*coverFileName)
		if err != nil {
			return nil, fmt.Errorf("given cover file '%s' does not exist", *coverFileName)
		}

		_, ok := supportedCoverMimeTypes[getMimeFromFileName(*coverFileName)]
		if !ok {
			return nil, fmt.Errorf("given cover file '%s' is not supported", *coverFileName)
		}
	} else if discoverMagicFiles && inputDirectory != nil {
		filter := func(file os.FileInfo) bool {
			if !file.IsDir() {
				if _, ok := artworkCoverFiles[strings.ToLower(file.Name())]; ok {
					fmt.Fprintf(output, "Info: applying magic cover file '%s'\n", file.Name())
					return true
				}
			}
			return false
		}

		var (
			fileNames []string
			err       error
		)
		if fileNames, err = directoryReader(fs, *inputDirectory, filter, true); err != nil {
			return nil, err
		}

		if len(fileNames) == 1 {
			coverFileName = &fileNames[0]
		}
	}

	if coverFileName == nil {
		return nil, nil
	}

	fileAbs, err := fs.Abs(*coverFileName)
	if err != nil {
		return nil, fmt.Errorf("can not get absolute path for cover file '%s', %v", *coverFileName, err)
	}

	return &fileAbs, nil
}

func getMimeFromFileName(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp3":
		return "audio/mpeg"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func getInterlaceFileName(
	fs fileSystemAbstraction,
	output io.Writer,
	interlaceFileName *string,
	filter fileInfoFilterFn,
	discoverMagicFiles bool,
	mediaFiles []string,
) (*string, error) {
	if discoverMagicFiles && interlaceFileName == nil {
		for _, f := range mediaFiles {
			name := filepath.Base(f)
			if name == magicInterlaceFileName {
				fmt.Fprintf(output, "Info: applying magic interlace file '%s'\n", f)
				interlaceFileName = stringToPtr(f)

				break
			}
		}
	}

	if interlaceFileName == nil {
		return nil, nil
	}

	inter, err := fs.Abs(*interlaceFileName)
	interlaceFileName = &inter

	if err != nil {
		return nil, fmt.Errorf("can not get absolute location of file '%s', %v", *interlaceFileName, err)
	}

	info, err := fs.Stat(*interlaceFileName)
	if err != nil {
		return nil, fmt.Errorf("given interlace file '%s' does not exist", *interlaceFileName)
	}

	if !filter(info) {
		return nil, fmt.Errorf("given media file '%s' is not a media file", *interlaceFileName)
	}

	return interlaceFileName, nil
}

func removeElementsFromStringList(list, elementsToRemove []string) []string {
	index := 0

loop:
	for _, elementInList := range list {
		for _, elementToRemove := range elementsToRemove {
			if elementToRemove == elementInList {
				continue loop
			}
		}
		list[index] = elementInList
		index++
	}

	return list[:index]
}

func getElementByIndex(output io.Writer, list []string, index int) (*string, error) {
	indexZeroBased := index - 1

	if index == 0 && len(list) > 0 {
		// assume the user wanted the first file
		indexZeroBased = 0

		fmt.Fprintln(output, "Info: file index '0' for copying metadata was specified, using first file")
	} else if indexZeroBased < 0 || indexZeroBased >= len(list) {
		return nil, fmt.Errorf("file index '%d' for copying metadata is invalid", indexZeroBased+1)
	}

	return &list[indexZeroBased], nil
}

func addInterlaceToStringList(list []string, interlace string) []string {
	if len(list) == 0 {
		return list
	}

	interlaced := make([]string, 0, len(list)*interlaceFilesScaleFactor)
	for _, f := range list {
		interlaced = append(interlaced, f, interlace)
	}

	return interlaced[:len(interlaced)-1]
}

func kvStringToTagMap(infoWriter io.Writer, kvp string) map[string]string {
	tags := make(map[string]string)

	// Set of known tags that is used to warn the caller if the tag specified
	// is not well-known
	knownTags := make(map[string]bool, len(id3v2.V23CommonIDs))
	for _, tagName := range id3v2.V23CommonIDs {
		knownTags[tagName] = true
	}

	// Very simple "key1=value1,key2=value2" parser
	for _, meta := range strings.Split(kvp, pairSeparator) {
		pairs := strings.Split(meta, valueSeparator)
		if len(pairs) != keyValuePairSize {
			fmt.Fprintf(infoWriter, "Warning: tag definition '%s' is not in the form 'key%svalue', ignoring\n", meta, valueSeparator)
			continue
		}

		tag, value := pairs[0], pairs[1]

		_, exist := knownTags[tag]
		if !exist {
			fmt.Fprintf(infoWriter, "Warning: tag '%s' is not a well-known tag, ignoring\n", tag)
			continue
		}

		tags[tag] = value
	}

	return tags
}
