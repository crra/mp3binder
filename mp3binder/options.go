//go:generate stringer -type=stage -trimprefix stage
package mp3binder

import (
	"errors"
	"fmt"
	"io"

	"github.com/bogem/id3v2/v2"
)

type stage int

const (
	stageInit stage = iota

	stageReadMetadata

	stageBind

	stageCopyMetadata
	stageAfterCopyMetadata

	stageBeforeWriteMetadata
	stageWriteMetadata

	// https://stackoverflow.com/questions/64178176/how-to-create-an-enum-and-iterate-over-it
	stageLastElement
)

const (
	defaultTrackNumber = "1"
	tagIdTrack         = "TRCK"
)

var (
	ErrTagNonStandard   = errors.New("non-standard tag")
	ErrNoTagsInTemplate = errors.New("no tags in template")
	ErrTagSkipCopying   = errors.New("ignoring tag for copying")
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
			template := j.metadata[index]
			if !template.HasFrames() {
				j.tagObserver("", "", ErrNoTagsInTemplate)
				return nil
			}

			for id := range template.AllFrames() {
				for _, f := range template.GetFrames(id) {
					switch ff := f.(type) {
					case id3v2.TextFrame:
						if id == tagIdTrack {
							j.tagObserver(id, "", ErrTagSkipCopying)
							continue
						}

						j.tagObserver(id, ff.Text, nil)
					case id3v2.PictureFrame:
						j.tagObserver(id, fmt.Sprintf("Image of type '%s'", ff.MimeType), nil)
					case id3v2.ChapterFrame:
						j.tagObserver(id, "", ErrTagSkipCopying)
						continue
					}

					j.tag.AddFrame(id, f)
				}
			}

			return nil
		}
	}
}

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
					j.tagObserver(id, "", ErrTagNonStandard)
					continue
				}

				j.tagObserver(id, value, nil)
				j.tag.AddFrame(id, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: value})
			}

			if _, ok := tags[tagIdTrack]; !ok {
				j.tag.AddFrame(tagIdTrack, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: defaultTrackNumber})
				j.tagObserver(tagIdTrack, defaultTrackNumber, nil)
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
