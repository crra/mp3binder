# About

_mp3binder_ is a simple command line utility for concatenating/join MP3 files without re-encoding.

Older mp3players e.g. stereos for children are not always capable of playing multiple files of an audio book in the correct order.

It is based on: http://www.dmulholl.com/dev/mp3cat.html and adds some more "batteries" like applying folder images (not yet implemented) and applying id3 tags.

# Usage

Files to be merged can be specified as a list of filenames:

`$ mp3binder one.mp3 two.mp3 three.mp3`

Alternatively, an entire directory of .mp3 files can be merged:

`$ mp3binder --dir /path/to/directory`

ID3 tags could be copied from the n-th input file:

`$ mp3binder -c 1 one.mp3 two.mp3 three.mp3`

ID3 tags could also be set manually. The name of the tag must be according to https://id3.org/id3v2.3.0#Declared_ID3v2_frames:

`$ mp3binder -c 1 -m "TRCK=42,TIT2=My sample title" one.mp3 two.mp3`
