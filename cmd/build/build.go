package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
)

// Using a simple golang program rather than a shell script to build binaries

type target struct {
	os   string
	arch string
}

type bundle struct {
	sourcePath string
	targetName string
}

var allTargets = []target{
	{os: "windows", arch: "amd64"},
	{os: "linux", arch: "amd64"},
	{os: "darwin", arch: "amd64"},
}

const (
	distFolder = "dist"
)

var bundles = []bundle{
	{sourcePath: "cmd/mp3binder/mp3binder.go", targetName: "mp3binder"},
}

var (
	productionLdFlags = []string{"-s", "-w"}
)

func getLatestGitTag(defaultValue string) string {
	cmd := exec.Command("git", "describe", "--tags")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return defaultValue
	}

	return strings.TrimSpace(string(output))
}

func isSandboxClean() bool {
	// https://stackoverflow.com/questions/3878624/how-do-i-programmatically-determine-if-there-are-uncommitted-changes
	commands :=
		[][]string{
			{"git", "update-index", "--refresh"},
			{"git", "diff-index", "--quiet", "HEAD", "--"},
		}

	for _, c := range commands {
		cmd := exec.Command(c[0], c[1:]...)
		err := cmd.Start()
		if err != nil {
			return false
		}
		err = cmd.Wait()
		if err != nil {
			return false
		}
	}

	return true
}

func getVersion() string {
	version := getLatestGitTag("untagged")

	if !isSandboxClean() {
		version = version + "-dirty"
	}

	return version
}

func main() {
	buildAll := flag.Bool("a", false, "Build all targets")
	buildProduction := flag.Bool("p", false, "Build for production")
	flag.Parse()

	buildTargets := []target{}
	if *buildAll {
		buildTargets = allTargets
	} else {
		buildTargets = []target{{os: runtime.GOOS, arch: runtime.GOARCH}}
	}

	err := os.RemoveAll(distFolder) // delete an entire directory
	if err != nil {
		log.Fatalf("Can't delete outputfolder, %v", err)
	}

	linkerVariables := map[string]string{}
	if *buildProduction {
		linkerVariables["main.version"] = getVersion()
	}

	var wg sync.WaitGroup
	for _, t := range buildTargets {
		for _, b := range bundles {
			wg.Add(1)
			go build(
				&wg,
				b,
				t,
				log.New(os.Stderr, fmt.Sprintf("%s/%s> ", t.os, t.arch), log.Flags()),
				linkerVariables,
				distFolder,
				*buildProduction,
			)
		}
	}
	wg.Wait()
}

func addFileToZip(zipWriter *zip.Writer, sourcePath, zipPath string) error {
	fileToZip, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func build(wg *sync.WaitGroup, b bundle, t target, l *log.Logger, variables map[string]string, outFolder string, isProductBuild bool) {
	defer wg.Done()
	l.Print("Start building")

	buildType := "develop"
	if isProductBuild {
		buildType = "release"
	}

	targetName := b.targetName
	if t.os == "windows" {
		targetName = targetName + ".exe"
	}

	outfile := path.Join(outFolder, buildType, t.os, targetName)
	args := []string{"build", "-o", outfile}

	// Linker flags
	if isProductBuild || len(variables) > 0 {
		ldflags := []string{}

		if isProductBuild {
			ldflags = append(ldflags, productionLdFlags...)
		}

		for k, v := range variables {
			ldflags = append(ldflags, fmt.Sprintf("-X \"%s=%s\"", k, v))
		}

		args = append(args, "-ldflags", strings.Join(ldflags, " "))
	}

	args = append(args, b.sourcePath)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOOS="+t.os, "GOARCH="+t.arch)

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Build failed, %v", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Build failed, %v", err)
	}

	if isProductBuild {
		zipFileName := path.Join(outFolder, buildType, t.os, fmt.Sprintf("%s-%s.zip", t.os, b.targetName))
		newZipFile, err := os.Create(zipFileName)
		if err != nil {
			log.Fatalf("Can't create zip file, %v", err)
		}
		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer zipWriter.Close()

		err = addFileToZip(zipWriter, outfile, targetName)
		if err != nil {
			log.Fatalf("Can't create zip file, %v", err)
		}
	}
}
