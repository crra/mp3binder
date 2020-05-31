# About

_mp3binder_ is a simple command line utility for concatenating/join MP3 files without re-encoding.

Older mp3players e.g. stereos for children are not always capable of playing multiple files of an audio book in the correct order.

It is based on: http://www.dmulholl.com/dev/mp3cat.html and adds some more "batteries" like applying id3 tags or determine the output filename based on a given folder or apply an interlace file automatically in a folder if the file is named "\_interlace.mp3".

# Usage

```
  -cover value
        use image file as artwork

  -d    prints debug information for each processing step

  -dir value
        directory of files to merge

  -f    overwrite an existing output file

  -interlace value
        interlace a spacer file (e.g. silence) between each input file

  -out value
        output filepath. Defaults to name of the folder of the first file provided

  -q    suppress info and warnings

  -tapply value
        apply id3v2 tags to output file.
        Takes the format 'key1=value,key2=value'.
        Keys should be from https://id3.org/id3v2.3.0#Declared_ID3v2_frames

  -tcopy value
        copy the ID3 metadata tag from the n-th input file, starting with 1

  -v    show version info
```

# Examples

Files to be merged can be specified as a list of filenames:

`$ mp3binder one.mp3 two.mp3 three.mp3`

Alternatively, an entire directory of .mp3 files can be merged:

`$ mp3binder --dir /path/to/directory`

ID3 tags could be copied from the n-th input file:

`$ mp3binder -tcopy 1 one.mp3 two.mp3 three.mp3`

ID3 tags could also be set manually. The name of the tag must be according to https://id3.org/id3v2.3.0#Declared_ID3v2_frames:

`$ mp3binder -tcopy 1 -tapply "TRCK=42,TIT2=My sample title" one.mp3 two.mp3`

# Silence between each tracks via interlace file

Based on: http://activearchives.org/wiki/Padding_an_audio_file_with_silence_using_sox

Create a silence track: `sox -n -r 44100 -c 2 silence.mp3 trim 0.0 3.0`

If the input material is FBR (fixed bit rate), generate the silence track with the same fixed bit rate using the '-C' option: `sox -n -r 44100 -c 2 -C 192 silence.mp3 trim 0.0 3.0`. (`file one.mp3` gives information about the current bit rate)

And apply: `mp3bind -interlace silence.mp3 01.mp3 02.mp3`
