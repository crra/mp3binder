package cli

import (
	"bufio"
	"errors"
	"fmt"
	fs2 "io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carolynvs/aferox"
	"github.com/crra/mp3binder/encoding/keyvalue"
	"github.com/crra/mp3binder/slice"
	"github.com/crra/mp3binder/value"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
)

// args is the cobra way of performing checks on the arguments before running                                                                                                                                                                                                                                                                                                                                                                                                                                                the application.
func (a *application) args(c *cobra.Command, args []string) error {
	if a.verbose {
		a.statusPrinter = newVerbosePrinter(a.status)
	}

	var err error
	a.language, err = language.Parse(a.languageStr)
	if err != nil {
		return fmt.Errorf("provided language '%s': %w", a.languageStr, ErrUnsupportedLanguage)
	}

	a.statusPrinter.language(a.language.String())

	// Treat an input file as list of arguments.
	// Any explicitly set argument has order priority over the input file argument.
	if a.inputFile != "" {
		argsFromInputFile, err := getInputFileAsList(a.fs, a.inputFile)
		if err != nil {
			return err
		}

		args = append(args, argsFromInputFile...)
	}

	mediaFiles, outputCandidateName, err := getMediaFilesFromArguments(a.fs, args)
	if err != nil {
		return err
	}

	a.outputPath, err = getOutputFile(a.fs, a.outputPath, a.overwrite, outputCandidateName)
	if err != nil {
		if errors.Is(err, ErrOutputFileExists) {
			return fmt.Errorf("use '--force' to overwrite: %w", err)
		}

		return err
	}

	a.mediaFiles = filterMediaFiles(mediaFiles, a.outputPath)

	if len(a.mediaFiles) == 0 {
		return ErrNoInput
	}

	if len(a.mediaFiles) < 2 {
		return ErrAtLeastTwo
	}

	a.statusPrinter.listInputFiles(a.mediaFiles, a.outputPath)

	a.coverFile, a.coverFileMimeType, err = lookupMimeType(getDiscoverableFile(a.fs, a.coverFile, a.noDiscovery, "cover", isAcceptedCoverFile, coverFiles))
	if err != nil {
		return err
	}

	a.statusPrinter.coverFile(a.coverFile)

	a.interlaceFile, err = getDiscoverableFile(a.fs, a.interlaceFile, a.noDiscovery, "interlace", isAcceptedInterlaceFile, interlaceFiles)
	if err != nil {
		return err
	}

	a.statusPrinter.interlaceFile(a.interlaceFile)

	if a.copyTagsFromIndex > 0 {
		if a.copyTagsFromIndex-1 >= len(a.mediaFiles) {
			return fmt.Errorf("index: '%d': %w", a.copyTagsFromIndex, ErrInvalidIndex)
		}

		a.statusPrinter.copyTagsFrom(a.mediaFiles[a.copyTagsFromIndex-1])
	}

	if a.applyTags != "" {
		tags, err := keyvalue.StringAsStringMap(a.applyTags)
		if err != nil {
			return err
		}

		for k, v := range tags {
			if v == "" {
				delete(a.tags, k)
				// keep the empty value in a.tags to remove the tag again when e.g. copying from an input file
			}

			if !a.verbose {
				if _, err := a.tagResolver.DescriptionFor(k); err != nil {
					fmt.Fprintf(a.status, "! Warning: the tag '%s' is not a well-known tag, but will be written\n", k)
				}
			}

			a.tags[k] = v
		}

		a.statusPrinter.tagsToApply(tags, a.tagResolver)
	}

	return nil
}

// isAcceptedMediaFile indicates if a file is accepted for joining.
func isAcceptedMediaFile(path string, skipInterlaceFiles bool) bool {
	// ignore the magic interlace files
	if skipInterlaceFiles && slice.Contains(interlaceFiles, strings.ToLower(filepath.Base(path))) {
		return false
	}

	return slice.Contains(mediaFileExtensions, strings.ToLower(filepath.Ext(path)))
}

// getInputFileAsList takes the content of an input file provides it as a list. An empty
// file is treated as an error.
func getInputFileAsList(fs aferox.Aferox, inputFile string) ([]string, error) {
	abs := fs.Abs(inputFile)
	exists, err := fs.Exists(abs)
	switch {
	case err != nil:
		return nil, err
	case !exists:
		return nil, fmt.Errorf("'%s': %w", abs, ErrFileNotFound)
	}

	isDir, err := fs.IsDir(abs)
	switch {
	case err != nil:
		return nil, err
	case isDir:
		return nil, fmt.Errorf("file is a directory '%s': %w", abs, ErrInvalidFile)
	}

	isEmpty, err := fs.IsEmpty(abs)
	switch {
	case err != nil:
		return nil, err
	case isEmpty:
		return nil, fmt.Errorf("file is empty '%s': %w", abs, ErrInvalidFile)
	}

	// File content as simple string array
	f, err := fs.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	args := []string{}

	for s.Scan() {
		args = append(args, s.Text())
	}

	return args, nil
}

// getMediaFilesFromArguments takes the program arguments and either accepts the argument as a file or if the argument
// is a directory, accepts the files contained in the directory.
func getMediaFilesFromArguments(fs aferox.Aferox, args []string) ([]mediaFile, string, error) {
	var files []mediaFile
	var outputFileCandidate string

	// no argument is provided, use the current directory and try to bind
	// the files that are listed here
	if len(args) == 0 {
		args = append(args, ".")
	}

	for _, arg := range args {
		filesFromParameter, candidate, err := getMediaFilesFromArgument(fs, arg)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return nil, "", fmt.Errorf("file: '%s': %w", arg, ErrFileNotFound)
			}

			return nil, "", err
		}

		if outputFileCandidate == "" && candidate != "" {
			outputFileCandidate = asOutputFile(candidate)
		}
		files = append(files, filesFromParameter...)
	}

	return files, outputFileCandidate, nil
}

// getMediaFilesFromArgument takes a program argument and either accepts the argument as a file or if the argument
// is a directory, accepts the files contained in the directory.
func getMediaFilesFromArgument(fs aferox.Aferox, arg string) ([]mediaFile, string, error) {
	arg = fs.Abs(arg)

	info, err := fs.Stat(arg)
	if err != nil {
		return nil, "", err
	}

	// regular file
	if !info.IsDir() {
		if isAcceptedMediaFile(arg, false) {
			return []mediaFile{{path: arg, explicitlySet: true}}, filepath.Base(filepath.Dir(arg)), nil
		}

		return nil, "", fmt.Errorf("media file '%s': %w", info.Name(), ErrInvalidFile)
	}

	// special case for root directories (e.g. removable media)
	candidateName := value.OrDefaultStr(info.Name(), rootDirectoryName)

	// files from a directory
	dirListing, err := fs.ReadDir(arg)
	if err != nil {
		return nil, "", err
	}

	var files []mediaFile
	for _, file := range dirListing {
		if file.IsDir() {
			continue
		}

		abs := fs.Abs(filepath.Join(arg, file.Name()))
		if !isAcceptedMediaFile(abs, true) {
			continue
		}

		files = append(files, mediaFile{path: abs, explicitlySet: false})
	}

	return files, candidateName, nil
}

// isAcceptedCoverFile returns true if the provided path points to a valid cover file.
func isAcceptedCoverFile(path string) bool {
	return slice.Contains(coverFileExtensions, strings.ToLower(filepath.Ext(path)))
}

// isAcceptedCoverFile returns true if the provided path points to a valid interlace file.
func isAcceptedInterlaceFile(path string) bool {
	return slice.Contains(mediaFileExtensions, strings.ToLower(filepath.Ext(path)))
}

// lookupMimeType accepts the returns of a 'filename/error' function and
// annotates the result with the mime type of the 'filename'.
func lookupMimeType(name string, err error) (string, string, error) {
	if err != nil {
		return name, "", err
	}

	return name, getMimeTypeOfImageByExtension(filepath.Ext(name)), nil
}

func getDiscoverableFile(fs aferox.Aferox, file string, noDiscovery bool, fileType string, accept func(string) bool, wellKnownFiles []string) (string, error) {
	// not set and no discovery
	if file == "" && noDiscovery {
		return "", nil
	}

	// explicitly set
	if file != "" {
		file = fs.Abs(file)

		info, err := fs.Stat(file)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return "", fmt.Errorf("%s file: '%s': %w", fileType, file, ErrFileNotFound)
			}

			return "", err
		}

		if info.IsDir() {
			return "", ErrInvalidFile
		}

		if !accept(file) {
			return "", fmt.Errorf("%s file '%s': %w", fileType, info.Name(), ErrInvalidFile)
		}

		return file, nil
	}

	// discover
	dir := fs.Abs(fs.Getwd())
	dirListing, err := fs.ReadDir(dir)
	if err != nil {
		return "", err
	}

	if found := slice.FirstEqual(slice.Map(dirListing, func(info fs2.FileInfo) string { return info.Name() }), wellKnownFiles, strings.ToLower); found != nil {
		return filepath.Join(dir, *found), nil
	}

	return "", nil
}

func getOutputFile(fs aferox.Aferox, outputPath string, overwrite bool, candidate string) (string, error) {
	if outputPath == "" {
		outputPath = candidate
	}

	outputPath = fs.Abs(outputPath)

	for {
		// Don't enforce the output file extension on the
		// first run the path may point to a directory where the
		// output file should be stored in.
		info, err := fs.Stat(outputPath)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				if hasOutputFileExtension(outputPath) {
					return outputPath, nil
				}

				outputPath = asOutputFile(outputPath)
				continue
			}
		}

		if info.IsDir() {
			name := info.Name()
			if name == "" {
				name = candidate
			}

			outputPath = asOutputFile(filepath.Join(outputPath, asOutputFile(name)))
			continue
		}

		if !overwrite {
			return "", fmt.Errorf("file: '%s': %w", outputPath, ErrOutputFileExists)
		}

		return outputPath, nil
	}
}

// hasOutputFileExtension takes a filename and checks it the output file extension is present.
func hasOutputFileExtension(fileName string) bool {
	return strings.ToLower(filepath.Ext(fileName)) == outputFileExtension
}

// asOutputFile takes a filename and converts it to an output file name.
func asOutputFile(fileName string) string {
	if !hasOutputFileExtension(fileName) {
		fileName += outputFileExtension
	}

	return fileName
}

const (
	partitionExplicitlySet = "explicitly"
	partitionDiscovered    = "discovered"
)

func mediaFilePartitionStr(m mediaFile) string {
	if m.explicitlySet {
		return partitionExplicitlySet
	}

	return partitionDiscovered
}

// filterMediaFiles performs various filters (e.g. removing duplicates when the sources:
// directory, command line arguments are mixed).
func filterMediaFiles(files []mediaFile, outputFile string) []string {
	if len(files) == 0 {
		return []string{}
	}

	seenFiles := make(map[string]struct{})
	preparedFiles := make([]mediaFile, 0, len(files))

	for _, f := range files {
		_, seen := seenFiles[f.path]
		switch {
		case f.path == outputFile:
			// the output file shall never be an input file
			continue
		case seen:
			// Allow duplicates for explicitly set files, but
			// filter duplicates for indirectly set files.
			// If a file was added explicitly and implicitly,
			// it will be filtered by partitioning (see after
			// this block)
			if !f.explicitlySet {
				continue
			}
		}

		seenFiles[f.path] = struct{}{}
		preparedFiles = append(preparedFiles, f)
	}

	partition := slice.Partition(preparedFiles, mediaFilePartitionStr)
	explicitlySet, _ := partition[partitionExplicitlySet]
	discovered, _ := partition[partitionDiscovered]

	// if there is no partition, just return the files
	if len(explicitlySet) == 0 || len(discovered) == 0 {
		return slice.Map(preparedFiles, slice.String[mediaFile])
	}

	// remove the explicitly set files from the discovered files
	orderedFiles := slice.UnionButIntersectionFromB(discovered, explicitlySet, slice.String[slice.PartitionResult[mediaFile]])

	// sort by the original index
	sort.Slice(orderedFiles, func(p, q int) bool {
		return orderedFiles[p].OriginalIndex < orderedFiles[q].OriginalIndex
	})

	return slice.Map(orderedFiles, slice.String[slice.PartitionResult[mediaFile]])
}

func getMimeTypeOfImageByExtension(ext string) string {
	switch strings.ToLower(ext) {
	default:
		fallthrough
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	}
}
