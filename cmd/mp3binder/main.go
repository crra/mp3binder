//+build !test

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/crra/mp3binder/ioext"
	"github.com/crra/mp3binder/mp3binder"
)

type osFileSystem struct{}

func (fs *osFileSystem) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (fs *osFileSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}
func (fs *osFileSystem) Abs(path string) (string, error) { return filepath.Abs(path) }

func main() {
	input, err := newUserInput(os.Args, os.Stderr, flag.ExitOnError)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if input.showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	informationWriter := ioutil.Discard
	if input.printInformation {
		informationWriter = os.Stdout
	}

	filesystem := &osFileSystem{}
	if err := run(input, informationWriter, filesystem); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input *userInput, informationWriter io.Writer, fs fileSystemAbstraction) error {
	var (
		err error
		j   *job
	)

	if j, err = newJobFromInput(fs, input, informationWriter); err != nil {
		return err
	}

	outFile, err := os.Create(j.outputFileName)
	if err != nil {
		return fmt.Errorf("can not open output file to write to, %v", err)
	}

	outFileCloser := ioext.OnceCloser(outFile)
	defer outFileCloser.Close()

	in := make([]io.Reader, len(j.files))

	for i, file := range j.files {
		var rc io.ReadCloser

		if rc, err = os.Open(file); err != nil {
			return fmt.Errorf("can not open media file for reading, %v", err)
		}

		defer rc.Close()

		in[i] = rc
	}

	var templateReader io.Reader

	if j.tagTemplateFile != nil {
		var templateReaderFile *os.File

		if templateReaderFile, err = os.Open(*j.tagTemplateFile); err != nil {
			return fmt.Errorf("can not open template media file for reading, %v", err)
		}
		defer templateReaderFile.Close()
		templateReader = templateReaderFile
	}

	var cover *mp3binder.Cover

	if j.coverFileName != nil {
		var coverFile *os.File

		if coverFile, err = os.Open(*j.coverFileName); err != nil {
			return fmt.Errorf("can not open media file for reading, %v", err)
		}
		defer coverFile.Close()

		cover = &mp3binder.Cover{
			MimeType: getMimeFromFileName(*j.coverFileName),
			Reader:   coverFile,
			Force:    input.coverFileName != nil,
		}
	}

	var progress = func(index int) {
		if len(j.files) > index {
			fmt.Fprintf(informationWriter, "- Processing: %s\n", filepath.Base(j.files[index]))
		}
	}

	if err := mp3binder.Bind(outFile, templateReader, j.tags, cover, progress, in...); err != nil {
		return err
	}

	return nil
}
