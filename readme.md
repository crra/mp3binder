# About

_mp3binder_ is a simple command line utility for concatenating/joining MP3 (formally: MPEG-1 Audio Layer III or MPEG-2 Audio Layer III) files without re-encoding.

Older mp3players e.g. stereos for children are not always capable of playing multiple files of an audio book in the correct order. Joining them into one file removes this limitation.

It is based on: http://www.dmulholl.com/dev/mp3cat.html and adds some more "batteries" like applying id3 tags or determine the output filename based on a given folder or apply an interlace file automatically in a folder if the file is named "\_interlace.mp3".

# Screenshot

![screenshot of the interface](doc/interface.png)

_Note: color added for clarity_

# Usage

```
mp3builder is a simple command line utility for concatenating/joining MP3 files without re-encoding.

Usage:
  mp3builder one.mp3 two.mp3 three.mp3 [flags]

Flags:
      --nomagic            ignores well-known files (e.g. folder.jpg)
      --cover string       use image file as artwork
      --verbose            prints verbose information for each processing step
      --force              overwrite an existing output file
      --interlace string   interlace a spacer file (e.g. silence) between each input file
      --output string      output filepath. Defaults to name of the folder of the first file provided
      --tapply string      apply id3v2 tags to output file.
                           Takes the format: 'key1=value,key2=value'.
                           Keys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames
      --tcopy int          copy the ID3 metadata tag from the n-th input file, starting with 1
  -h, --help               help for mp3builder
  -v, --version            version for mp3builder
```

# Examples

Files to be merged can be specified as a list of filenames:

`$ mp3binder one.mp3 two.mp3 three.mp3`

Alternatively, an entire directory of .mp3 files can be merged:

- `$ mp3binder` (the program uses the current directory)
- `$ mp3binder .` (the program reads the directory)
- `$ mp3binder *.mp3` (the shell expands the files)

ID3 tags can be copied from the n-th input file:

`$ mp3binder --tcopy 1 one.mp3 two.mp3 three.mp3`

ID3 tags could also be set manually. The name of the tag must be according to https://id3.org/id3v2.3.0#Declared_ID3v2_frames:

`$ mp3binder --tcopy 1 --tapply 'TRCK=42,TIT2="My sample title"' one.mp3 two.mp3`

# Silence between each tracks via interlace file

Based on: http://activearchives.org/wiki/Padding_an_audio_file_with_silence_using_sox

Create a silence track: `sox -n -r 44100 -c 2 silence.mp3 trim 0.0 3.0`

If the input material is FBR (fixed bit rate), generate the silence track with the same fixed bit rate using the '-C' option: `sox -n -r 44100 -c 2 -C 192 silence.mp3 trim 0.0 3.0`. The shell command `file one.mp3` gives information about the bit rate of a file.

And apply: `mp3bind --interlace silence.mp3 01.mp3 02.mp3`

# Build instructions

Building is always more complex then just calling `build` (e.g. adding version information into the binary or naming the binary or optimize the binary by stripping debug information). Instead of a `Makefile`, a `Taskfile.yml` is used that holds the instructions for [Task](https://taskfile.dev). 'Task' is not mandatory but simplifies the workflow. Once installed ([instructions](https://taskfile.dev/#/installation)), 'Task' provides an executable `task` that can be called with custom actions.

For example cross compilation for multiple platforms is achieved with `task build-all`, and for the current platform with `task build`. Plain go would be: `go build .\cmd\mp3builder\` without build time optimizations and settings (e.g. version information).

The `Taskfile.yml` gives good hints which commands and options are executed if the developer don't want to use `task`. In the end 'Task' it's just a simple task runner (collection of commands).
