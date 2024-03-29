package cli

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/crra/mp3binder/io/rewindingreader"
	"github.com/crra/mp3binder/mp3binder"
	"github.com/crra/mp3binder/slice"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// run is the cobra way of running the application.
func (a *application) run(c *cobra.Command, _ []string) error {
	if a.interlaceFile != "" {
		a.mediaFiles = slice.Interlace(a.mediaFiles, a.interlaceFile)
		a.copyTagsFromIndex = slice.IndexAfterInterlace(len(a.mediaFiles), a.copyTagsFromIndex-1)

		a.statusPrinter.listMediaFilesAfterInterlace(a.mediaFiles)
	}

	// final output file
	output, err := a.fs.Create(a.outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	// audio only file
	audioOnlyFile, err := a.fs.TempFile(filepath.Dir(a.outputPath), "")
	if err != nil {
		return err
	}
	defer func() {
		audioOnlyFile.Close()
		a.fs.Remove(audioOnlyFile.Name())
	}()

	// inputs
	inputs, openFilesCloser, err := openFilesOnce(a.fs, a.mediaFiles)
	defer openFilesCloser()
	if err != nil {
		return err
	}

	// options for the bind process
	options, optionsCloser, err := a.bindingOptions()
	defer optionsCloser()
	if err != nil {
		return err
	}

	// bind
	err = a.binder.Bind(a.parent, output, audioOnlyFile, inputs, options...)
	if err != nil {
		_ = a.fs.Remove(a.outputPath)
		return err
	}

	return nil
}

// unCamel takes a string following the CamelCase notation and separates the string
// by spaces on word boundaries.
func unCamel(s string) string {
	b := &strings.Builder{}

	for i, r := range s {
		if i > 0 {
			if unicode.IsUpper(r) {
				b.WriteRune(' ')
			}
		}

		b.WriteRune(r)
	}

	return b.String()
}

// openFilesOnce opens a list of filenames and returns
func openFilesOnce(fs afero.Fs, files []string) ([]io.Reader, func(), error) {
	input := make([]io.Reader, len(files))
	openedFiles := make(map[string]afero.File)

	close := func() {
		for name := range openedFiles {
			if file, ok := openedFiles[name]; ok && file != nil {
				file.Close()
			}
		}
	}

	for i, name := range files {
		if f, ok := openedFiles[name]; ok {
			input[i] = f
			continue
		}

		f, err := fs.Open(name)
		if err != nil {
			close()
			return input, func() {}, err
		}

		input[i] = rewindingreader.New(f)
		openedFiles[name] = f
	}

	return input, close, nil
}

func titleFromString(language language.Tag, fileName string) string {
	fileName = filepath.Base(fileName)
	return cases.Title(language).String(strings.TrimSuffix(fileName, path.Ext(fileName)))
}

// bindingOptions returns configuration options for the bind method based on the user input.
func (a *application) bindingOptions() ([]any, func(), error) {
	options := []any{}

	var coverFile io.Closer
	closer := func() {
		if coverFile != nil {
			coverFile.Close()
		}
	}

	// status visitors
	if a.verbose {
		options = append(options,
			mp3binder.ActionVisitor(a.statusPrinter.actionObserver),
			mp3binder.BindVisitor(a.statusPrinter.newBindObserver(a.mediaFiles)),
			mp3binder.TagApplyVisitor(a.statusPrinter.newTagObserver(a.tags)),
		)
	}

	// copy tags
	if a.copyTagsFromIndex > 0 {
		options = append(options, mp3binder.TagCopyVisitor(
			a.statusPrinter.newTagCopyObserver(a.mediaFiles[a.copyTagsFromIndex-1])))
	}

	// chapter
	if !a.noChapters {
		// contains titles for chapters filled by the id3v2 title of the input file
		chapterTitles := make([]string, len(a.mediaFiles))

		// Extract the title tag from each media file by collecting the titles
		// from the file's metadata.
		options = append(options, mp3binder.MetadataVisitor(func(index int, tags map[string]string) {
			chapterTitles[index] = tags[tagTitle]

			if chapterTitles[index] == "" {
				chapterTitles[index] = titleFromString(a.language, a.mediaFiles[index])
			}
		}))

		// Enable chapters and register a function that provides the name of the chapter.
		options = append(options, mp3binder.Chapters(func(index, chapterIndex int) (bool, string) {
			chapterTitle := fmt.Sprintf("Chapter %d", chapterIndex)

			if title := chapterTitles[index]; title != "" {
				chapterTitle = title
			}

			return (a.interlaceFile == "") || (a.mediaFiles[index] != a.interlaceFile), chapterTitle
		}))
	}

	// cover file
	if a.coverFile != "" {
		coverFile, err := a.fs.Open(a.coverFile)
		if err != nil {
			return []any{}, closer, err
		}

		options = append(options, mp3binder.Cover(a.coverFileMimeType, coverFile))
	}

	// copy metadata
	if a.copyTagsFromIndex > 0 {
		options = append(options, mp3binder.CopyMetadataFrom(a.copyTagsFromIndex-1, ErrNoTagsInTemplate))
	}

	// apply metadata
	options = append(options, mp3binder.ApplyTextMetadata(func(previous map[string]string) map[string]string {
		// If the title is not explicitly set (empty erases) or copied from a file from the index,
		// use the folder name of the first input file.
		_, explicitlySet := a.tags[tagTitle]
		_, copied := previous[tagTitle]

		if !explicitlySet && !copied {
			a.tags[tagTitle] = titleFromString(a.language, filepath.Dir(a.mediaFiles[0]))
		}

		return a.tags
	}))

	return options, closer, nil
}
