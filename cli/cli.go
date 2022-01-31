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
	"golang.org/x/text/language"
)

var (
	ErrNoInput             = errors.New("no input files specified")
	ErrAtLeastTwo          = errors.New("at least two files are required")
	ErrInvalidFile         = errors.New("invalid file")
	ErrFileNotFound        = errors.New("file not found")
	ErrOutputFileExists    = errors.New("output file is already existing")
	ErrInvalidIndex        = errors.New("the provided index is invalid")
	ErrTagNonStandard      = errors.New("non-standard tag")
	ErrUnsupportedLanguage = errors.New("unsupported language")
	ErrNoTagsInTemplate    = errors.New("no tags in template")
)

const (
	flagNoDiscovery   = "nodiscovery"
	flagNoChapters    = "nochapters"
	flagCover         = "cover"
	flagVerbose       = "verbose"
	flagOverwrite     = "force"
	flagInterlaceFile = "interlace"
	flagOutputFile    = "output"
	flagApplyTags     = "tapply"
	flagCopyTags      = "tcopy"
	flagLanguageStr   = "lang"
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

const tagTitle = "TIT2"

type statusPrinter interface {
	language(language string)
	listMediaFilesAfterInterlace(mediaFiles []string)
	listInputFiles(mediaFiles []string, outputFile string)
	coverFile(file string)
	interlaceFile(file string)
	copyTagsFrom(file string)
	tagsToApply(tags map[string]string, tagResolver tagResolver)

	actionObserver(stage, action string)
	newBindObserver(mediaFiles []string) func(index int)
	newTagCopyObserver(copyFilename string) func(tag, value string, err error)
	newTagObserver(tags map[string]string) func(tag, value string, err error)
}

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

	fs            aferox.Aferox
	cwd           string
	status        io.Writer
	statusPrinter statusPrinter

	parent context.Context

	noDiscovery       bool
	noChapters        bool
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

		status:        status,
		statusPrinter: &discardingPrinter{},

		fs:  aferox.NewAferox(cwd, fs),
		cwd: cwd,

		tags: map[string]string{
			tagEncoderSoftware: fmt.Sprintf("%s %s", name, version),
			tagIdTrack:         defaultTrackNumber,
		},
		languageStr: userLocale,
	}

	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s file1.mp3 file2.mp3", name),
		Example: fmt.Sprintf("Calling '%[1]s' with no parameters is equivalent to: '%[1]s *.mp3'", name),
		Version: version,
		Short:   fmt.Sprintf("%s joins multiple mp3 files into one without re-encoding", name),
		Long:    fmt.Sprintf("%s joins multiple MP3 files into one without re-encoding. It writes ID3v2 tags and chapters for each file.", name),

		SilenceErrors: true,
		SilenceUsage:  true,

		Args: app.args,
		RunE: app.run,
	}

	cmd.SetOutput(status)
	app.command = cmd

	f := cmd.Flags()
	f.SortFlags = false // prefer the order defined by the code

	f.BoolVar(&app.noDiscovery, flagNoDiscovery, app.noDiscovery, "no discovery for well-known files (e.g. cover.jpg)")
	f.BoolVar(&app.noChapters, flagNoChapters, app.noChapters, "does not write chapters for bounded files")
	f.StringVar(&app.coverFile, flagCover, app.coverFile, "use image file as artwork")
	f.BoolVar(&app.verbose, flagVerbose, app.verbose, "prints verbose information for each processing step")
	f.BoolVar(&app.overwrite, flagOverwrite, app.overwrite, "overwrite an existing output file")
	f.StringVar(&app.interlaceFile, flagInterlaceFile, app.interlaceFile, "interlace a spacer file (e.g. silence) between each input file")
	f.StringVar(&app.outputPath, flagOutputFile, app.outputPath, "output filepath. Defaults to name of the folder of the first file provided")
	f.StringVar(&app.applyTags, flagApplyTags, app.applyTags, "apply id3v2 tags to output file.\nTakes the format: 'key1=value,key2=value'.\nKeys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames")
	f.IntVar(&app.copyTagsFromIndex, flagCopyTags, app.copyTagsFromIndex, "copy the ID3 metadata tag from the n-th input file, starting with 1")
	f.StringVar(&app.languageStr, flagLanguageStr, app.languageStr, "ISO-639 language string used during string manipulation (e.g. uppercase)")

	return app
}
