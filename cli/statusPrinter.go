package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
)

type discardingPrinter struct {
	statusPrinter
}

func (d *discardingPrinter) language(language string)                                    {}
func (d *discardingPrinter) listInputFiles(mediaFiles []string, outputFile string)       {}
func (d *discardingPrinter) listMediaFilesAfterInterlace(mediaFiles []string)            {}
func (d *discardingPrinter) coverFile(file string)                                       {}
func (d *discardingPrinter) interlaceFile(file string)                                   {}
func (d *discardingPrinter) copyTagsFrom(file string)                                    {}
func (d *discardingPrinter) tagsToApply(tags map[string]string, tagResolver tagResolver) {}

func (d *discardingPrinter) actionObserver(stage, action string) {}
func (d *discardingPrinter) newBindObserver(mediaFiles []string) func(index int) {
	return func(index int) {}
}

func (d *discardingPrinter) newTagCopyObserver(copyFilename string) func(tag, value string, err error) {
	return func(tag, value string, err error) {}
}

func (d *discardingPrinter) newTagObserver(tags map[string]string) func(tag, value string, err error) {
	return func(tag, value string, err error) {}
}

type verbosePrinter struct {
	output io.Writer
}

func newVerbosePrinter(output io.Writer) statusPrinter {
	return &verbosePrinter{
		output: output,
	}
}

func (p *verbosePrinter) language(language string) {
	fmt.Fprintf(p.output, "The following language will be used: '%s'\n", language)
}

func (p *verbosePrinter) listInputFiles(mediaFiles []string, outputPath string) {
	p.list(mediaFiles, fmt.Sprintf("The following files will be 'bound' as: '%s'", outputPath))
}

func (p *verbosePrinter) listMediaFilesAfterInterlace(mediaFiles []string) {
	p.list(mediaFiles, "Files to bind after applying the interlace file:")
}

//
func (p *verbosePrinter) list(mediaFiles []string, title string) {
	padding := len(strconv.Itoa(len(mediaFiles)))

	fmt.Fprintln(p.output, title)
	format := fmt.Sprintf("- %%%[1]dd: %%s\n", padding)
	for i, f := range mediaFiles {
		fmt.Fprintf(p.output, format, i+1, f)
	}
}

func (p *verbosePrinter) coverFile(coverFile string) {
	if coverFile != "" {
		fmt.Fprintf(p.output, "The following file will be used as cover: '%s'\n", coverFile)
	}
}

func (p *verbosePrinter) interlaceFile(interlaceFile string) {
	if interlaceFile != "" {
		fmt.Fprintf(p.output, "The following file will be used as interlace: '%s'\n", interlaceFile)
	}
}

func (p *verbosePrinter) copyTagsFrom(mediaFile string) {
	fmt.Fprintf(p.output, "Id3v2 tags will be copied from file: '%s'\n", mediaFile)
}

func (p *verbosePrinter) tagsToApply(tags map[string]string, tagResolver tagResolver) {
	fmt.Fprintln(p.output, "The following id3v2 tags will be applied:")

	for k := range tags {
		v := tags[k]
		description, err := tagResolver.DescriptionFor(k)
		switch {
		case v == "" && err != nil:
			fmt.Fprintf(p.output, "- Not well-known '%s': will be removed if present in the output\n", k)
		case v == "" && err == nil:
			fmt.Fprintf(p.output, "- %s (%s): will be removed if present in the output\n", description, k)
		case err != nil:
			fmt.Fprintf(p.output, "- Not well-known '%s': %s\n", k, v)
		default:
			fmt.Fprintf(p.output, "- %s (%s): %s\n", description, k, v)
		}
	}
}

func (p *verbosePrinter) actionObserver(stage, action string) {
	fmt.Fprintf(p.output, "Processing stage: '%s' and action: '%s'\n", unCamel(stage), action)
}

func (p *verbosePrinter) newBindObserver(mediaFiles []string) func(index int) {
	return func(index int) {
		fmt.Fprintf(p.output, "- Binding: '%s'\n", filepath.Base(mediaFiles[index]))
	}
}

func (p *verbosePrinter) newTagCopyObserver(copyFilename string) func(tag, value string, err error) {
	return func(tag, value string, err error) {
		switch {
		case err != nil && errors.Is(err, ErrNoTagsInTemplate):
			fmt.Fprintf(p.output, "! Warning: there are not id3v2 tags in file: '%s'\n", copyFilename)
		case err != nil:
			fmt.Fprintf(p.output, "! Unhandled warning during tag processing: %v\n", err)
		default:
			fmt.Fprintf(p.output, "- Copying tag: '%s' with value '%s'\n", tag, value)
		}
	}
}

func (p *verbosePrinter) newTagObserver(tags map[string]string) func(tag, value string, err error) {
	return func(tag, value string, err error) {
		switch {
		case err != nil && errors.Is(err, ErrTagNonStandard):
			fmt.Fprintf(p.output, "! Warning: tag '%s' with value '%s' is not well-known, but will be written\n", tag, tags[tag])
		case err != nil:
			fmt.Fprintf(p.output, "! Unhandled warning during tag processing: %v\n", err)
		case value == "":
			fmt.Fprintf(p.output, "- Removing tag: '%s'\n", tag)
		default:
			fmt.Fprintf(p.output, "- Adding tag: '%s' with value '%s'\n", tag, value)
		}
	}
}
