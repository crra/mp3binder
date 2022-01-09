package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	fs2 "io/fs"
	"path/filepath"

	"github.com/crra/mp3binder/slice"

	"github.com/carolynvs/aferox"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	ErrNoInput      = errors.New("no input files specified")
	ErrAtLeastTwo   = errors.New("at least two files are required")
	ErrIsDir        = errors.New("input file is directory")
	ErrInvalidFile  = errors.New("invalid file")
	ErrFileNotFound = errors.New("file not found")
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
	mediaFileExtensions = []string{".mp3"}
	coverFileExtensions = []string{".jpg", ".png"}
	coverFileNames      = []string{"cover", "folder", "album"}
	coverFiles          = slice.ConcatStr(coverFileNames, coverFileExtensions)
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
	outputFile    string
	applyTags     string
	copyTags      int
	mediaFiles    []string

	command *cobra.Command
}

type Service interface {
	Execute() error
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
	f.StringVar(&app.outputFile, flagOutputFile, app.outputFile, "output filepath. Defaults to name of the folder of the first file provided")
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
func isAcceptedMediaFile(path string) bool {
	return slice.Contains(mediaFileExtensions, filepath.Ext(path))
}

// getMediaFilesFromArguments takes the program arguments and either accepts the argument as a file or if the argument
// is a directory, accepts the files contained in the directory.
func getMediaFilesFromArguments(fs aferox.Aferox, args []string) ([]string, error) {
	var files []string

	for _, arg := range args {
		filesFromParameter, err := getMediaFilesFromArgument(fs, arg)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return nil, fmt.Errorf("file: '%s': %w", arg, ErrFileNotFound)
			}

			return nil, err
		}

		files = append(files, filesFromParameter...)
	}

	return files, nil
}

// getMediaFilesFromArgument takes a program argument and either accepts the argument as a file or if the argument
// is a directory, accepts the files contained in the directory.
func getMediaFilesFromArgument(fs aferox.Aferox, arg string) ([]string, error) {
	abs := fs.Abs(arg)

	info, err := fs.Stat(abs)
	if err != nil {
		return nil, err
	}

	// regular file
	if !info.IsDir() {
		if isAcceptedMediaFile(abs) {
			return []string{abs}, nil
		}

		return nil, fmt.Errorf("media file '%s': %w", info.Name(), ErrInvalidFile)
	}

	// files from a directory
	dirListing, err := fs.ReadDir(arg)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, file := range dirListing {
		if file.IsDir() {
			continue
		}

		abs := fs.Abs(filepath.Join(abs, file.Name()))
		if !isAcceptedMediaFile(abs) {
			continue
		}

		files = append(files, abs)
	}

	return files, nil
}

// isAcceptedCoverFile returns true if the provided path points to a valid cover file.
func isAcceptedCoverFile(path string) bool {
	return slice.Contains(coverFileExtensions, filepath.Ext(path))
}

// getCoverFile returns the coverfile either explicitly provided or found in the current directory.
func getCoverFile(fs aferox.Aferox, noDiscovery bool, cover string) (string, error) {
	// not set and no discovery
	if cover == "" && noDiscovery {
		return "", nil
	}

	// explicitly set
	if cover != "" {
		cover = fs.Abs(cover)

		info, err := fs.Stat(cover)
		if err != nil {
			if errors.Is(err, fs2.ErrNotExist) {
				return "", fmt.Errorf("cover: '%s': %w", cover, ErrFileNotFound)
			}

			return "", err
		}

		if info.IsDir() {
			return "", ErrInvalidFile
		}

		if !isAcceptedCoverFile(cover) {
			return "", fmt.Errorf("cover file '%s': %w", info.Name(), ErrInvalidFile)
		}

		return cover, nil
	}

	// discover
	dir := fs.Abs(fs.Getwd())
	dirListing, err := fs.ReadDir(dir)
	if err != nil {
		return "", err
	}

	if found := slice.FirstEqual(slice.Map(dirListing, func(info fs2.FileInfo) string { return info.Name() }), coverFiles); found != nil {
		return filepath.Join(dir, *found), nil
	}

	return "", nil
}

// args is the cobra way of performing checks on the arguments before running                                                                                                                                                                                                                                                                                                                                                                                                                                                the application.
func (a *application) args(c *cobra.Command, args []string) error {
	mediaFiles, err := getMediaFilesFromArguments(a.fs, args)
	if err != nil {
		return err
	}

	if len(mediaFiles) == 0 {
		return ErrNoInput
	}

	if len(mediaFiles) < 2 {
		return ErrAtLeastTwo
	}

	a.mediaFiles = mediaFiles

	a.coverFile, err = getCoverFile(a.fs, a.noDiscovery, a.coverFile)
	if err != nil {
		return err
	}

	return nil
}
