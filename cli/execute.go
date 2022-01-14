package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/crra/mp3binder/mp3binder"
	"github.com/crra/mp3binder/slice"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// run is the cobra way of running the application.
func (a *application) run(c *cobra.Command, _ []string) error {
	if a.interlaceFile != "" {
		a.mediaFiles = slice.Interlace(a.mediaFiles, a.interlaceFile)
		a.copyTagsFromIndex = slice.IndexAfterInterlace(len(a.mediaFiles), a.copyTagsFromIndex)

		if a.verbose {
			padding := len(strconv.Itoa(len(a.mediaFiles)))

			fmt.Fprintln(a.status, "Files to bind after applying the interlace file:")
			format := fmt.Sprintf("> %%%[1]dd: %%s\n", padding)
			for i, f := range a.mediaFiles {
				fmt.Fprintf(a.status, format, i+1, f)
			}
		}
	}

	// output
	output, err := a.fs.Create(a.outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	// inputs
	input, close, err := openFilesOnce(a.fs, a.mediaFiles)
	defer close()
	if err != nil {
		return err
	}

	var options []mp3binder.Option

	if a.verbose {
		options = append(options, mp3binder.ActionObserver(func(stage string, action string) {
			fmt.Fprintf(a.status, "Processing stage: '%s' and action: '%s'\n", unCamel(stage), action)
		}))

		options = append(options, mp3binder.BindObserver(func(index int) {
			fmt.Fprintf(a.status, "Binding: '%s'\n", filepath.Base(a.mediaFiles[index]))
		}))

		options = append(options, mp3binder.TagObserver(func(tag string, err error) {
			switch {
			case errors.Is(err, mp3binder.ErrNonStandardTag):
				fmt.Fprintf(a.status, "Warning: tag '%s' with value '%s' is not a well-known tag, ignoring\n", tag, a.tags[tag])
			default:
				fmt.Fprintf(a.status, "Adding tag: '%s' with value '%s'\n", tag, a.tags[tag])
			}
		}))
	}

	// cover file
	if a.coverFile != "" {
		f, err := a.fs.Open(a.coverFile)
		if err != nil {
			return err
		}

		defer f.Close()
		options = append(options, mp3binder.Cover(a.coverFileMimeType, f))
	}

	// copy metadata
	if a.copyTagsFromIndex > 0 {
		options = append(options, mp3binder.CopyMetadataFrom(a.copyTagsFromIndex))
	}

	// apply metadata
	if len(a.tags) > 0 {
		options = append(options, mp3binder.ApplyMetadata(a.tags))
	}

	// bind
	err = mp3binder.Bind(a.parent, output, input, options...)
	if err != nil {
		_ = a.fs.Remove(a.outputPath)
		return err
	}

	return nil
}

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

func openFilesOnce(fs afero.Fs, files []string) ([]io.ReadSeeker, func(), error) {
	input := make([]io.ReadSeeker, len(files))
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

		input[i] = f
		openedFiles[name] = f
	}

	return input, close, nil
}
