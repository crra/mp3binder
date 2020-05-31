package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/bogem/id3v2"
	"github.com/crra/mp3binder/flagext"
	"github.com/crra/mp3binder/ioext"
	"github.com/dmulholl/mp3lib"
)

var version = "unversioned"

const (
	extOfMp3               = ".mp3"
	magicInterlaceFilename = "_interlace.mp3"
	tagCover               = "APIC"

	pairSeparator  = ","
	valueSeparator = "="
)

var (
	artworkCoverFiles       = make(map[string]struct{})
	supportedCoverMimeTypes = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
	}
)

func init() {
	artworkNames := []string{"cover", "folder", "artwork"}

	for _, ext := range supportedCoverMimeTypes {
		for _, name := range artworkNames {
			artworkCoverFiles[name+ext] = struct{}{}
		}
	}
}

// context holds the application state
type context struct {
	showInformationDuringProcessing bool
	forceOverwrite                  bool
	outputFilename                  *string

	inputDirectory *string
	inputFiles     []string

	interlaceFilename         *string
	coverFilename             *string
	forceCover                bool
	copyMetadataFromFileIndex *int
	id3tags                   *string
	metadataForOutputFile     map[string]string

	mediaFiles []string
}

func (c *context) String() string {
	return c.StringWithPrefix("")
}

func (c *context) StringWithPrefix(p string) string {
	var str strings.Builder
	str.WriteString(p)
	str.WriteString(" Context {\n")

	// General
	str.WriteString(p)
	str.WriteString("  - Output filename: ")
	if c.outputFilename == nil {
		str.WriteString("!not set")
	} else {
		str.WriteString(*c.outputFilename)
	}
	str.WriteString("\n")

	str.WriteString(p)
	str.WriteString(fmt.Sprintf("  - Cover filename (force:%t): ", c.forceCover))
	if c.coverFilename == nil {
		str.WriteString("!not set")
	} else {
		str.WriteString(*c.coverFilename)
	}
	str.WriteString("\n")

	// Files
	str.WriteString(fmt.Sprintf("%s  - Files: (%d)\n", p, len(c.mediaFiles)))
	for i, f := range c.mediaFiles {
		str.WriteString(fmt.Sprintf("%s    #%02d %s\n", p, i, f))
	}
	str.WriteString(p)

	// Metadata
	str.WriteString(fmt.Sprintf("%s  - Metadata: (%d)\n", p, len(c.metadataForOutputFile)))
	for k, v := range c.metadataForOutputFile {
		str.WriteString(fmt.Sprintf("%s    #%s %s\n", p, k, v))
	}
	str.WriteString(p)

	str.WriteString(" }\n")
	return str.String()
}

// "railway-oriented programming" looks a lot like a chain of middlewares or httpHandlerFuncs
type pipelineFunc func(*context) error
type pipelineFuncInterceptor func(*context, pipelineFunc) error
type pipeline []pipelineFunc

func run(pipeline []pipelineFunc, c *context) error {
	for _, f := range pipeline {
		err := f(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func runWithInterceptor(pipeline []pipelineFunc, c *context, interceptor pipelineFuncInterceptor) error {
	for _, f := range pipeline {
		err := interceptor(c, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func newInterceptor(w io.Writer) pipelineFuncInterceptor {
	i := 0
	return func(c *context, f pipelineFunc) error {
		funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		fmt.Fprintf(w, "%02d: %s\n", i, funcName)
		fmt.Printf("%v\n", c.StringWithPrefix(">"))
		err := f(c)
		fmt.Fprintf(w, "%v\n", c.StringWithPrefix("<"))
		fmt.Fprintln(w, "")
		i++
		if err != nil {
			return err
		}
		return nil
	}
}

func main() {
	ctx := &context{
		metadataForOutputFile: make(map[string]string),
	}

	// Flags
	flagVersion := flag.Bool("v", false, "show version info")
	flag.BoolVar(&ctx.forceOverwrite, "f", false, "overwrite an existing output file")
	flagQuiet := flag.Bool("q", false, "suppress info and warnings")
	flagDebug := flag.Bool("d", false, "prints debug information for each processing step")

	// Values
	flag.Var(flagext.NewStringPtrFlag(&ctx.outputFilename), "out", "output filepath. Defaults to name of the folder of the first file provided")
	flag.Var(flagext.NewStringPtrFlag(&ctx.inputDirectory), "dir", "directory of files to merge")
	flag.Var(flagext.NewStringPtrFlag(&ctx.interlaceFilename), "interlace", "interlace a spacer file (e.g. silence) between each input file")
	flag.Var(flagext.NewStringPtrFlag(&ctx.coverFilename), "cover", "use image file as artwork")
	flag.Var(flagext.NewStringPtrFlag(&ctx.id3tags), "tapply", "apply id3v2 tags to output file.\nTakes the format 'key1=value,key2=value'.\nKeys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames")
	flag.Var(flagext.NewIntPtrFlag(&ctx.copyMetadataFromFileIndex), "tcopy", "copy the ID3 metadata tag from the n-th input file, starting with 1")

	flag.Parse()
	ctx.showInformationDuringProcessing = !*flagQuiet
	ctx.inputFiles = flag.Args()

	if flagVersion != nil && *flagVersion == true {
		fmt.Println(version)
		return
	}

	p := pipeline{
		collectFilesFromCommandline,
		collectFilesFromDirectory,
		setCoverfile,
		setInterlaceFile,
		mustHaveMediaFiles,
		removePossibleInterlaceFileFromFiles,
		setOutputFileName,
		removePossibleOutputFileFromFiles,
		interlaceFiles,
		bindFiles,
		setMetadataCopyIndex,
		collectID3TagsFromCommandline,
		applyMetadata,
	}

	var err error
	if *flagDebug == true {
		err = runWithInterceptor(p, ctx, newInterceptor(os.Stdout))
	} else {
		err = run(p, ctx)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func collectFilesFromDirectory(c *context) error {
	if c.inputDirectory == nil {
		return nil
	}

	if !filepath.IsAbs(*c.inputDirectory) {
		abs, err := filepath.Abs(*c.inputDirectory)
		if err != nil {
			return fmt.Errorf("can not get absolute path for directory '%s', %v", *c.inputDirectory, err)
		}
		c.inputDirectory = &abs
	}

	files := []string{}

	dirContent, err := ioutil.ReadDir(*c.inputDirectory)
	if err != nil {
		return fmt.Errorf("can not read media files from directory, %v", err)
	}
	for _, file := range dirContent {
		if !file.IsDir() && strings.ToLower(filepath.Ext(file.Name())) == extOfMp3 {
			files = append(files, filepath.Join(*c.inputDirectory, file.Name()))
		}
	}

	if len(files) == 0 && c.showInformationDuringProcessing {
		fmt.Println("Warning: input directory contains no media files")
	}

	c.mediaFiles = append(c.mediaFiles, files...)

	return nil
}

func validateInputFile(f string) error {
	_, err := os.Stat(f)
	if err != nil {
		return fmt.Errorf("given media file '%s' does not exist", f)
	}

	ext := strings.ToLower(filepath.Ext(f))
	if ext != extOfMp3 {
		return fmt.Errorf("given media file '%s' is not a media file", f)
	}

	return nil
}

func collectFilesFromCommandline(c *context) error {
	for _, f := range c.inputFiles {
		file, err := filepath.Abs(f)
		if err != nil {
			return fmt.Errorf("can get absolute location of file '%s', %v", f, err)
		}

		err = validateInputFile(file)
		if err != nil {
			return err
		}

		c.mediaFiles = append(c.mediaFiles, file)
	}

	return nil
}

func mustHaveMediaFiles(c *context) error {
	if len(c.mediaFiles) == 0 {
		return fmt.Errorf("no media files for processing available")
	} else if len(c.mediaFiles) == 1 {
		return fmt.Errorf("only one media file for processing available")
	}

	return nil
}

func setInterlaceFile(c *context) error {
	if c.interlaceFilename == nil {
		for _, f := range c.mediaFiles {
			name := filepath.Base(f)
			if name == magicInterlaceFilename {
				if c.showInformationDuringProcessing {
					fmt.Printf("Info: found magic interlace file '%s', applying\n", f)
				}
				c.interlaceFilename = &f
				break
			}
		}

		if c.interlaceFilename == nil {
			return nil
		}
	}

	interlaceFilename, err := filepath.Abs(*c.interlaceFilename)
	if err != nil {
		return fmt.Errorf("can not get absolute location of file '%s', %v", interlaceFilename, err)
	}

	err = validateInputFile(interlaceFilename)
	if err != nil {
		return err
	}

	c.interlaceFilename = &interlaceFilename

	return nil
}

func interlaceFiles(c *context) error {
	if c.interlaceFilename == nil {
		return nil
	}

	interlaced := make([]string, 0, len(c.mediaFiles)*2)
	for _, f := range c.mediaFiles {
		interlaced = append(interlaced, f)
		interlaced = append(interlaced, *c.interlaceFilename)
	}

	c.mediaFiles = interlaced[:len(interlaced)-1]

	return nil
}

func setOutputFileName(c *context) error {
	var outputFilename string

	if c.outputFilename == nil {
		if len(c.mediaFiles) == 0 {
			return fmt.Errorf("output filename not set and no files provided that could be used to deduce filename")
		}

		pathOfFirstFile := filepath.Dir(c.mediaFiles[0])
		nameOfFolder := filepath.Base(pathOfFirstFile)
		outputFilename = filepath.Join(pathOfFirstFile, nameOfFolder+extOfMp3)
		c.outputFilename = &outputFilename
	}

	abs, err := filepath.Abs(*c.outputFilename)
	if err != nil {
		return fmt.Errorf("can get absolute location of file '%s', %v", *c.outputFilename, err)
	}

	_, err = os.Stat(abs)
	if err == nil && c.forceOverwrite == false {
		return fmt.Errorf("file already exists '%s', use force (-f) to overwrite", *c.outputFilename)
	}

	c.outputFilename = &abs

	return nil
}

func removePossibleInterlaceFileFromFiles(c *context) error {
	if c.interlaceFilename == nil {
		return nil
	}

	files, ok := removeStringFromList(c.mediaFiles, *c.interlaceFilename)
	if ok {
		c.mediaFiles = files
	}
	return nil
}

func removePossibleOutputFileFromFiles(c *context) error {
	files, ok := removeStringFromList(c.mediaFiles, *c.outputFilename)
	if ok {
		c.mediaFiles = files
	}
	return nil
}

func removeStringFromList(list []string, entry string) ([]string, bool) {
	newList := make([]string, 0, len(list))

	// There is no 'filter' in golang: https://github.com/robpike/filter
	// and no generics in go1
	for _, e := range list {
		if e == entry {
			continue
		}
		newList = append(newList, e)
	}

	return newList, len(list) != len(newList)
}

func bindFiles(c *context) error {
	outFile, err := os.Create(*c.outputFilename)
	if err != nil {
		return fmt.Errorf("can not open output file to write to, %v", err)
	}
	outFileCloser := ioext.OnceCloser(outFile)
	defer outFileCloser.Close()

	bitrates := make(map[int]struct{})
	var framesCount uint32
	var bytesCount uint32

	for _, file := range c.mediaFiles {
		inFile, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("can not open media file for reading, %v", err)
		}
		defer inFile.Close()

		for i := 0; true; i++ {
			frame := mp3lib.NextFrame(inFile)
			if frame == nil {
				break
			}

			// Skip the first frame if it's a VBR header.
			if i == 0 && (mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame)) {
				continue
			}

			bitrates[frame.BitRate] = struct{}{}

			_, err := outFile.Write(frame.RawBytes)
			if err != nil {
				return fmt.Errorf("can not write media file content to new file, %v", err)
			}
			framesCount++
			bytesCount += uint32(len(frame.RawBytes))
		}
	}

	outFileCloser.Close()
	if err != nil {
		return fmt.Errorf("error closing new file, %v", err)
	}

	// VBR header
	if len(bitrates) != 1 {
		err := addXingHeader(*c.outputFilename, framesCount, bytesCount)
		if err != nil {
			return fmt.Errorf("can not update vbr header in output file, %v", err)
		}
	}

	return nil
}

func addXingHeader(file string, totalFrames, totalBytes uint32) error {
	shell, err := ioutil.TempFile(filepath.Dir(file), "*")
	if err != nil {
		return fmt.Errorf("can not create temporay file, %v", err)
	}
	shellCloser := ioext.OnceCloser(shell)
	defer shellCloser.Close()

	xingHeader := mp3lib.NewXingHeader(totalFrames, totalBytes)

	// header + content
	_, err = shell.Write(xingHeader.RawBytes)
	if err != nil {
		return fmt.Errorf("can not write to temporary file, %v", err)
	}

	inputFile, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("can not open media file for reading, %v", err)
	}
	inputFileCloser := ioext.OnceCloser(inputFile)
	defer inputFileCloser.Close()

	_, err = io.Copy(shell, inputFile)
	if err != nil {
		return fmt.Errorf("can not open file for writing, %v", err)
	}

	err = shellCloser.Close()
	if err != nil {
		return fmt.Errorf("can not close temporary file, %v", err)
	}
	err = inputFileCloser.Close()
	if err != nil {
		return fmt.Errorf("can not close media file, %v", err)
	}

	err = os.Rename(shell.Name(), file)
	if err != nil {
		return fmt.Errorf("can not replace media file with temporary file, %v", err)
	}

	return nil
}

func setMetadataCopyIndex(c *context) error {
	if c.copyMetadataFromFileIndex == nil {
		return nil
	}

	indexZeroBased := *c.copyMetadataFromFileIndex - 1
	if *c.copyMetadataFromFileIndex == 0 {
		// assume the user wanted the first file
		indexZeroBased = 0
		if c.showInformationDuringProcessing {
			fmt.Println("info: file index '0' for copying metadata was specified, using first file")
		}
	} else if indexZeroBased < 0 || indexZeroBased >= len(c.mediaFiles) {
		return fmt.Errorf("file index '%d' for copying metadata is greater than the available files", indexZeroBased+1)
	}

	c.copyMetadataFromFileIndex = &indexZeroBased

	return nil
}

func collectID3TagsFromCommandline(c *context) error {
	if c.id3tags == nil {
		return nil
	}

	// Set of known tags that is used to warn the caller if the tag specified
	// is not well-known
	knownTags := make(map[string]bool, len(id3v2.V23CommonIDs))
	for _, tagName := range id3v2.V23CommonIDs {
		knownTags[tagName] = true
	}

	// Very simple "key1=value1,key2=value2" parser
	for _, meta := range strings.Split(*c.id3tags, pairSeparator) {
		pairs := strings.Split(meta, valueSeparator)
		if len(pairs) != 2 {
			if c.showInformationDuringProcessing {
				fmt.Printf("Warning: tag definition '%s' is not in the form key%svalue, ignoring\n", meta, valueSeparator)
			}
			continue
		}

		tag, value := pairs[0], pairs[1]

		_, exist := knownTags[tag]
		if !exist {
			if c.showInformationDuringProcessing {
				fmt.Printf("Warning: tag '%s' is not a well-known tag, ignoring\n", tag)
			}
			continue
		}

		c.metadataForOutputFile[tag] = value
	}

	return nil
}

func applyMetadata(c *context) error {
	if c.copyMetadataFromFileIndex == nil && len(c.metadataForOutputFile) == 0 && c.coverFilename == nil {
		return nil
	}

	tag, err := id3v2.Open(*c.outputFilename, id3v2.Options{Parse: false})
	if err != nil {
		return fmt.Errorf("can not read id3 information from media file, %v", err)
	}
	defer tag.Close()

	frames := make(map[string]id3v2.Framer)

	if c.copyMetadataFromFileIndex != nil {
		masterTag, err := id3v2.Open(c.mediaFiles[*c.copyMetadataFromFileIndex], id3v2.Options{Parse: true})
		if err != nil {
			return fmt.Errorf("can not read id3 information from media file, %v", err)
		}
		defer masterTag.Close()

		// from master file
		for id := range masterTag.AllFrames() {
			frames[id] = masterTag.GetLastFrame(id)
		}
	}

	// frames from commandline
	for id, value := range c.metadataForOutputFile {
		frames[id] = &id3v2.TextFrame{Encoding: tag.DefaultEncoding(), Text: value}
	}

	if c.coverFilename != nil {
		_, ok := frames[tagCover]
		if !ok || c.forceCover {
			cover, err := ioutil.ReadFile(*c.coverFilename)
			if err != nil {
				return fmt.Errorf("can not read cover file '%s', %v", *c.coverFilename, err)
			}

			pic := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    getMimeFromFilename(*c.coverFilename),
				PictureType: id3v2.PTFrontCover,
				Description: "Front cover",
				Picture:     cover,
			}
			tag.AddAttachedPicture(pic)
		} else {
			if c.showInformationDuringProcessing {
				fmt.Println("Info: output file already has a cover file, ignoring magic file")
			}
		}
	}

	// Apply all
	for id, f := range frames {
		tag.AddFrame(id, f)
	}

	err = tag.Save()
	if err != nil {
		return fmt.Errorf("can not write id3 tags to tile, %v", err)
	}

	return nil
}

func getMimeFromFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp3":
		return "audio/mpeg"
	case ".png":
		return "image/png"
	case ".jpeg":
		fallthrough
	case ".jpg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func setCoverfile(c *context) error {
	if c.coverFilename != nil {
		_, err := os.Stat(*c.coverFilename)
		if err != nil {
			return fmt.Errorf("given cover file '%s' does not exist", *c.coverFilename)
		}

		_, ok := supportedCoverMimeTypes[getMimeFromFilename(*c.coverFilename)]
		if !ok {
			return fmt.Errorf("given cover file '%s' is not supported", *c.coverFilename)
		}

		c.forceCover = true
	} else if c.inputDirectory != nil {
		dirContent, err := ioutil.ReadDir(*c.inputDirectory)
		if err != nil {
			return fmt.Errorf("can not files from directory, %v", err)
		}
		for _, file := range dirContent {
			if file.IsDir() {
				continue
			}

			if _, ok := artworkCoverFiles[strings.ToLower(file.Name())]; ok {
				file := filepath.Join(*c.inputDirectory, file.Name())
				if c.showInformationDuringProcessing {
					fmt.Printf("Info: found magic cover file '%s', applying\n", file)
				}
				c.coverFilename = &file
				break
			}
		}

		if c.coverFilename == nil {
			return nil
		}
	}

	abs, err := filepath.Abs(*c.coverFilename)
	if err != nil {
		return fmt.Errorf("can not get absolute path for cover file '%s', %v", *c.coverFilename, err)
	}
	c.coverFilename = &abs

	return nil
}
