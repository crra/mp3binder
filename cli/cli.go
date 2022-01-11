package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	fs2 "io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/crra/mp3binder/slice"
	"github.com/crra/mp3binder/value"

	"github.com/carolynvs/aferox"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	ErrNoInput          = errors.New("no input files specified")
	ErrAtLeastTwo       = errors.New("at least two files are required")
	ErrInvalidFile      = errors.New("invalid file")
	ErrFileNotFound     = errors.New("file not found")
	ErrOutputFileExists = errors.New("output file is already existing")
)

const (
	flagNoDiscovery   = "nomagic"
	flagCover         = "cover"
	flagDebug         = "debug"
	flagOverwrite     = "force"
	flagInterlaceFile = "interlace"
	flagOutputFile    = "output"
	flagApplyTags     = "tapply"
	flagCopyTags      = "tcopy"
)

var (
	outputFileExtension = ".mp3"
	mediaFileExtensions = []string{".mp3"}
	coverFileExtensions = []string{".jpg", ".png"}
	coverFileNames      = []string{"cover", "folder", "album"}
	coverFiles          = slice.ConcatStr(coverFileNames, coverFileExtensions)
	interlaceFiles      = []string{"interlace.mp3", "_interlace.mp3"}
)

type application struct {
	name    string
	version string

	fs     aferox.Aferox
	cwd    string
	status io.Writer

	log    logr.Logger
	parent context.Context

	noDiscovery   bool
	coverFile     string
	debug         bool
	overwrite     bool
	interlaceFile string
	outputPath    string
	applyTags     string
	copyTags      int
	mediaFiles    []string

	command *cobra.Command
}

// Service describes the cli service.
type Service interface {
	// Execute executes the service.
	Execute() error
}

type mediaFile struct {
	path          string
	explicitlySet bool
}

func (m mediaFile) String() string {
	return m.path
}

func New(parent context.Context, name, version string, log logr.Logger, status io.Writer, fs afero.Fs, cwd string) Service {
	app := &application{
		parent:  parent,
		name:    name,
		version: version,
		log:     log,
		status:  status,
		fs:      aferox.NewAferox(cwd, fs),
		cwd:     cwd,
	}

	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s one.mp3 two.mp3 three.mp3", name),
		Version: version,
		Short:   fmt.Sprintf("%s joins multiple mp3 files into one", name),
		Long:    fmt.Sprintf("%s is a simple command line utility for concatenating/join MP3 files without re-encoding.", name),

		SilenceErrors: true,
		SilenceUsage:  true,

		Args: app.args,
		RunE: app.run,
	}

	cmd.SetOutput(status)
	f := cmd.Flags()
	f.SortFlags = false // prefer the order defined by the code

	f.BoolVar(&app.noDiscovery, flagNoDiscovery, app.noDiscovery, "ignores well-known files (e.g. folder.jpg)")
	f.StringVar(&app.coverFile, flagCover, app.coverFile, "use image file as artwork")
	f.BoolVar(&app.debug, flagDebug, app.debug, "prints debug information for each processing step")
	f.BoolVar(&app.overwrite, flagOverwrite, app.overwrite, "overwrite an existing output file")
	f.StringVar(&app.interlaceFile, flagInterlaceFile, app.interlaceFile, "interlace a spacer file (e.g. silence) between each input file")
	f.StringVar(&app.outputPath, flagOutputFile, app.outputPath, "output filepath. Defaults to name of the folder of the first file provided")
	f.StringVar(&app.applyTags, flagApplyTags, app.applyTags, "apply id3v2 tags to output file.\nTakes the format: 'key1=value,key2=value'.\nKeys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames")
	f.IntVar(&app.copyTags, flagCopyTags, app.copyTags, "copy the ID3 metadata tag from the n-th input file, starting with 1")

	app.command = cmd

	return app
}

// Execute executes the application.
func (a *application) Execute() error {
	return a.command.Execute()
}

// run is the cobra way of running the application.
func (a *application) run(c *cobra.Command, _ []string) error {
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

	if a.debug {
		padding := len(strconv.Itoa(len(a.mediaFiles)))

		format := fmt.Sprintf("%%%[1]dd: %%s\n", padding)
		for i, f := range a.mediaFiles {
			fmt.Fprintf(a.status, format, i, f)
		}
	}

	a.coverFile, err = getDiscoverableFile(a.fs, a.coverFile, a.noDiscovery, "cover", isAcceptedCoverFile, coverFiles)
	if err != nil {
		return err
	}

	a.interlaceFile, err = getDiscoverableFile(a.fs, a.interlaceFile, a.noDiscovery, "interlace", isAcceptedInterlaceFile, interlaceFiles)
	if err != nil {
		return err
	}

	return nil
}
