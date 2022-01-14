//go:generate stringer -type=stage -trimprefix stage
package mp3binder

import (
	"errors"
	"io"

	"github.com/bogem/id3v2"
)

type stage int

const (
	stageInit stage = iota

	stageCopyMetadata
	stageAfterCopyMetadata
	stageBeforeWriteMetadata
	stageWriteMetadata

	stageBind

	// https://stackoverflow.com/questions/64178176/how-to-create-an-enum-and-iterate-over-it
	stageLastElement
)

const (
	defaultTrackNumber = "1"
	tagIdTrack         = "TRCK"
)

type (
	jobProcessor func(*job) error
	Option       func() (stage, string, jobProcessor)
)

func ActionObserver(f stageObserver) Option {
	return func() (stage, string, jobProcessor) {
		const action = "stage observer"
		return stageInit, action, func(j *job) error {
			j.stageObserver = f

			return nil
		}
	}
}

func BindObserver(f bindObserver) Option {
	return func() (stage, string, jobProcessor) {
		return stageInit, "bind observer", func(j *job) error {
			j.bindObserver = f

			return nil
		}
	}
}

func TagObserver(f tagObserver) Option {
	return func() (stage, string, jobProcessor) {
		return stageInit, "tag observer", func(j *job) error {
			j.tagObserver = f

			return nil
		}
	}
}

func CopyMetadataFrom(index int) Option {
	return func() (stage, string, jobProcessor) {
		return stageCopyMetadata, "copy metadata", func(j *job) error {
			tagFromTemplate, err := id3v2.ParseReader(j.input[index], id3v2.Options{Parse: true})
			if err != nil {
				return err
			}

			for id := range tagFromTemplate.AllFrames() {
				if id == tagIdTrack {
					continue
				}

				j.tag.AddFrame(id, tagFromTemplate.GetLastFrame(id))
			}

			return nil
		}
	}
}

var ErrNonStandardTag = errors.New("non-standard tag")

func ApplyMetadata(tags map[string]string) Option {
	knownTags := make(map[string]bool, len(id3v2.V23CommonIDs))
	for _, tagName := range id3v2.V23CommonIDs {
		knownTags[tagName] = true
	}

	return func() (stage, string, jobProcessor) {
		return stageBeforeWriteMetadata, "applying metadata", func(j *job) error {
			for id, value := range tags {
				_, exist := knownTags[id]
				if !exist {
					j.tagObserver(id, ErrNonStandardTag)
					continue
				}

				j.tag.AddFrame(id, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: value})
			}

			if _, ok := tags[tagIdTrack]; !ok {
				j.tag.AddFrame(tagIdTrack, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: defaultTrackNumber})
			}

			return nil
		}
	}
}

const (
	coverType = "Front cover"
)

func Cover(mimeType string, r io.Reader) Option {
	return func() (stage, string, jobProcessor) {
		return stageAfterCopyMetadata, "adding cover", func(j *job) error {
			data, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			j.tag.AddAttachedPicture(id3v2.PictureFrame{
				Encoding:    j.tag.DefaultEncoding(),
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: coverType,
				Picture:     data,
			})

			return nil
		}
	}
}
