package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/crra/mp3binder/slice"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	ErrNoInput          = errors.New("no input files specified")
	ErrAtLeastTwo       = errors.New("at least two files are required")
	ErrInvalidFile      = errors.New("invalid file")
	ErrFileNotFound     = errors.New("file not found")
	ErrOutputFileExists = errors.New("output file is already existing")
	ErrInvalidIndex     = errors.New("the provided index is invalid")
	ErrTagNonStandard   = errors.New("non-standard tag")
)

const (
	flagNoDiscovery   = "nomagic"
	flagCover         = "cover"
	flagVerbose       = "verbose"
	flagOverwrite     = "force"
	flagInterlaceFile = "interlace"
	flagOutputFile    = "output"
	flagApplyTags     = "tapply"
	flagCopyTags      = "tcopy"
)

var (
	outputFileExtension = ".mp3"
	mediaFileExtensions = []string{".mp3"}
	coverFileExtensions = []string{".jpg", ".jpeg", ".png"}
	coverFileNames      = []string{"cover", "folder", "album"}
	coverFiles          = slice.ConcatStr(coverFileNames, coverFileExtensions)
	interlaceFiles      = []string{"interlace.mp3", "_interlace.mp3"}

	rootDirectoryName = "root" + outputFileExtension
)

// Service describes the cli service.
type Service interface {
	// Execute executes the service.
	Execute() error
}

type binder interface {
	Bind(context.Context, io.WriteSeeker, io.ReadWriteSeeker, []io.ReadSeeker, ...any) error
}

type tagResolver interface {
	DescriptionFor(string) (string, error)
}

type application struct {
	name        string
	version     string
	binder      binder
	tagResolver tagResolver

	fs     aferox.Aferox
	cwd    string
	status io.Writer

	parent context.Context

	noDiscovery       bool
	coverFile         string
	coverFileMimeType string
	verbose           bool
	overwrite         bool
	interlaceFile     string
	outputPath        string
	applyTags         string
	languageStr       string
	language          language.Tag
	copyTagsFromIndex int // NOTE: starts on '1' rathen than '0'
	mediaFiles        []string
	tags              map[string]string

	command *cobra.Command
}

// Execute executes the application.
func (a *application) Execute() error {
	return a.command.Execute()
}

type mediaFile struct {
	path          string
	explicitlySet bool
}

func (m mediaFile) String() string {
	return m.path
}

const (
	tagEncoderSoftware = "TSSE"
	tagIdTrack         = "TRCK"
	defaultTrackNumber = "1"
)

func New(parent context.Context, name, version string, status io.Writer, fs afero.Fs, cwd string, binder binder, tagResolver tagResolver, userLocale string) Service {
	app := &application{
		parent:      parent,
		name:        name,
		version:     version,
		binder:      binder,
		tagResolver: tagResolver,
		log:         log,
		status:      status,
		fs:          aferox.NewAferox(cwd, fs),
		cwd:         cwd,
		tags: map[string]string{
			tagEncoderSoftware: fmt.Sprintf("%s %s", name, version),
			tagIdTrack:         defaultTrackNumber,
		},
		languageStr: userLocale,
	}

	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s file1.mp3 file2.mp3", name),
		Example: fmt.Sprintf("Calling '%[1]s' with no parameters is equivalent to: '%[1]s *.mp3 --nomagic'", name),
		Version: version,
		Short:   fmt.Sprintf("%s joins multiple mp3 files into one", name),
		Long:    fmt.Sprintf("%s is a simple command line utility for concatenating/joining MP3 files without re-encoding.", name),

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
	f.BoolVar(&app.verbose, flagVerbose, app.verbose, "prints verbose information for each processing step")
	f.BoolVar(&app.overwrite, flagOverwrite, app.overwrite, "overwrite an existing output file")
	f.StringVar(&app.interlaceFile, flagInterlaceFile, app.interlaceFile, "interlace a spacer file (e.g. silence) between each input file")
	f.StringVar(&app.outputPath, flagOutputFile, app.outputPath, "output filepath. Defaults to name of the folder of the first file provided")
	f.StringVar(&app.applyTags, flagApplyTags, app.applyTags, "apply id3v2 tags to output file.\nTakes the format: 'key1=value,key2=value'.\nKeys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames")
	f.IntVar(&app.copyTagsFromIndex, flagCopyTags, app.copyTagsFromIndex, "copy the ID3 metadata tag from the n-th input file, starting with 1")

	app.command = cmd

	return app
}
