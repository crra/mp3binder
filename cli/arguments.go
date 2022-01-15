package cli

import (
	"errors"
	"fmt"
	fs2 "io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/carolynvs/aferox"
	"github.com/crra/mp3binder/encoding/keyvalue"
	"github.com/crra/mp3binder/slice"
	"github.com/crra/mp3binder/value"
	"github.com/spf13/cobra"
)

// args is the cobra way of performing checks on the arguments before running                                                                                                                                                                                                                                                                                                                                                                                                                                                the application.
func (a *application) args(c *cobra.Command, args []string) error {
	mediaFiles, outputCandidateName, err := getMediaFilesFromArguments(a.fs, args)
	if err != nil {
		return err
	}

	a.outputPath, err = getOutputFile(a.fs, a.outputPath, a.overwrite, outputCandidateName)
	if err != nil {
		if errors.Is(err, ErrOutputFileExists) {
			return fmt.Errorf("%w, use '--force' to overwrite", err)
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

	if a.verbose {
		padding := len(strconv.Itoa(len(a.mediaFiles)))

		fmt.Fprintf(a.status, "The following files will be 'bound' as: '%s'\n", a.outputPath)
		format := fmt.Sprintf("> %%%[1]dd: %%s\n", padding)
		for i, f := range a.mediaFiles {
			fmt.Fprintf(a.status, format, i+1, f)
		}
	}

	a.coverFile, a.coverFileMimeType, err = lookupMimeType(getDiscoverableFile(a.fs, a.coverFile, a.noDiscovery, "cover", isAcceptedCoverFile, coverFiles))
	if err != nil {
		return err
	}

	if a.coverFile != "" {
		if a.verbose {
			fmt.Fprintf(a.status, "The following file will be used as cover: '%s'\n", a.coverFile)
		}
	}

	a.interlaceFile, err = getDiscoverableFile(a.fs, a.interlaceFile, a.noDiscovery, "interlace", isAcceptedInterlaceFile, interlaceFiles)
	if err != nil {
		return err
	}

	if a.verbose && a.interlaceFile != "" {
		fmt.Fprintf(a.status, "The following file will be used as interlace: '%s'\n", a.interlaceFile)
	}

	if a.copyTagsFromIndex > 0 {
		// zero based
		a.copyTagsFromIndex -= 1

		if a.copyTagsFromIndex >= len(a.mediaFiles) {
			return fmt.Errorf("index: '%d': %w", a.copyTagsFromIndex+1, ErrInvalidIndex)
		}

		if a.verbose {
			fmt.Fprintf(a.status, "The id3tags will be copied from file: '%s'\n", a.mediaFiles[a.copyTagsFromIndex])
		}
	}

	if a.applyTags != "" {
		a.tags, err = keyvalue.StringAsStringMap(a.applyTags)
		if err != nil {
			return err
		}
		if a.verbose {
			fmt.Fprintln(a.status, "The following id3tags will be applied:")
			for k := range a.tags {
				fmt.Fprintf(a.status, "> %s: %s\n", k, a.tags[k])
			}
		}
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

func hasOutputFileExtension(fileName string) bool {
	return strings.ToLower(filepath.Ext(fileName)) == outputFileExtension
}

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
