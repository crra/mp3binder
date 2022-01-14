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

	a.outputPath, err = getOutputFile(a.fs, a.outputPath, a.overwrite, asOutputFile(outputCandidateName))
	if err != nil {
		if errors.Is(err, ErrOutputFileExists) {
			return fmt.Errorf("%w, use '--force' to overwrite", err)
		}

		return err
	}

	a.mediaFiles = removeDuplicatesIfSourceMixed(mediaFiles, a.outputPath)

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

	a.coverFile, err = getDiscoverableFile(a.fs, a.coverFile, a.noDiscovery, "cover", isAcceptedCoverFile, coverFiles)
	if err != nil {
		return err
	}

	if a.coverFile != "" {
		if a.verbose {
			fmt.Fprintf(a.status, "The following file be used as cover: '%s'\n", a.coverFile)
		}

		a.coverFileMimeType = getMimeTypeForExtension(filepath.Ext(a.coverFile))
	}

	a.interlaceFile, err = getDiscoverableFile(a.fs, a.interlaceFile, a.noDiscovery, "interlace", isAcceptedInterlaceFile, interlaceFiles)
	if err != nil {
		return err
	}

	if a.verbose && a.interlaceFile != "" {
		fmt.Fprintf(a.status, "The following file be used as interlace: '%s'\n", a.interlaceFile)
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
	if skipInterlaceFiles && slice.Contains(interlaceFiles, filepath.Base(path)) {
		return false
	}

	return slice.Contains(mediaFileExtensions, filepath.Ext(path))
}

// getMediaFilesFromArguments takes the program arguments and either accepts the argument as a file or if the argument
// is a directory, accepts the files contained in the directory.
func getMediaFilesFromArguments(fs aferox.Aferox, args []string) ([]mediaFile, string, error) {
	var files []mediaFile
	var outputFileCandidate string

	for _, arg := range args {
		filesFromParameter, candidate, err := getMediaFilesFromArgument(fs, arg)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return nil, "", fmt.Errorf("file: '%s': %w", arg, ErrFileNotFound)
			}

			return nil, "", err
		}

		if outputFileCandidate == "" && candidate != "" {
			outputFileCandidate = candidate
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
	candidateName := value.OrDefaultStr(info.Name(), "root")

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
	return slice.Contains(coverFileExtensions, filepath.Ext(path))
}

// isAcceptedCoverFile returns true if the provided path points to a valid interlace file.
func isAcceptedInterlaceFile(path string) bool {
	return slice.Contains(mediaFileExtensions, filepath.Ext(path))
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
		info, err := fs.Stat(outputPath)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return outputPath, nil
			}
		}

		if info.IsDir() {
			name := info.Name()
			if name == "" {
				name = candidate
			}

			outputPath = filepath.Join(outputPath, candidate)
			continue
		}

		if !overwrite {
			return "", fmt.Errorf("file: '%s': %w", outputPath, ErrOutputFileExists)
		}

		return outputPath, nil
	}
}

func asOutputFile(fileName string) string {
	if filepath.Ext(fileName) != outputFileExtension {
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

func removeDuplicatesIfSourceMixed(files []mediaFile, outputCandidate string) []string {
	if len(files) == 0 {
		return []string{}
	}

	// remove output candidate if discovered
	for i, f := range files {
		if f.explicitlySet == false && f.path == outputCandidate {
			copy(files[i:], files[i+1:])
			files = files[:len(files)-1]
			break
		}
	}

	partition := slice.Partition(files, mediaFilePartitionStr)
	explicitlySet, _ := partition[partitionExplicitlySet]
	discovered, _ := partition[partitionDiscovered]

	// if there is no partition, just return the files
	if len(explicitlySet) == 0 || len(discovered) == 0 {
		return slice.Map(files, slice.String[mediaFile])
	}

	// remove the explicitly set files from the discovered files
	orderedFiles := slice.UnionButIntersectionFromB(discovered, explicitlySet, slice.String[slice.PartitionResult[mediaFile]])

	// sort by the original index
	sort.Slice(orderedFiles, func(p, q int) bool {
		return orderedFiles[p].OriginalIndex < orderedFiles[q].OriginalIndex
	})

	return slice.Map(orderedFiles, slice.String[slice.PartitionResult[mediaFile]])
}

func getMimeTypeForExtension(ext string) string {
	switch strings.ToLower(ext) {
	default:
		fallthrough
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	}
}
