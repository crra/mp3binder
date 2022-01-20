//go:generate stringer -type=stage -trimprefix stage
package mp3binder

import (
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

	stageBeforeWriteMetadata
	stageWriteMetadata

	// https://stackoverflow.com/questions/64178176/how-to-create-an-enum-and-iterate-over-it
	stageLastElement
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

func TagCopyObserver(f tagCopyObserver) Option {
	return func() (stage, string, jobProcessor) {
		return stageInit, "tag copy observer", func(j *job) error {
			j.tagCopyObserver = f

			return nil
		}
	}
}

func CopyMetadataFrom(index int, errNoTagsInTemplate error) Option {
	return func() (stage, string, jobProcessor) {
		return stageCopyMetadata, "copy metadata", func(j *job) error {
			template := j.metadata[index]
			if !template.HasFrames() {
				j.tagCopyObserver("", "", errNoTagsInTemplate)
				return nil
			}

			for id := range template.AllFrames() {
				for _, f := range template.GetFrames(id) {
					switch ff := f.(type) {
					case id3v2.TextFrame:
						j.tagCopyObserver(id, ff.Text, nil)
					case id3v2.PictureFrame:
						j.tagCopyObserver(id, fmt.Sprintf("Image of type '%s'", ff.MimeType), nil)
					case id3v2.ChapterFrame:
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
	return func() (stage, string, jobProcessor) {
		return stageBeforeWriteMetadata, "applying metadata", func(j *job) error {
			for id, value := range tags {
				description, err := j.tagResolver.DescriptionFor(id)
				if err != nil {
					j.tagObserver(id, "", fmt.Errorf("tag '%s': %w", id, err))
				} else {
					j.tagObserver(fmt.Sprintf("%s (%s)", description, id), value, nil)
				}

				if value == "" {
					continue
				}

				j.tag.AddFrame(id, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: value})
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
		return stageBeforeWriteMetadata, "adding cover", func(j *job) error {
			frontCoverPicture, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			j.tagObserver(coverType, mimeType, nil)

			j.tag.AddAttachedPicture(id3v2.PictureFrame{
				Encoding:    j.tag.DefaultEncoding(),
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: coverType,
				Picture:     frontCoverPicture,
			})

			return nil
		}
	}
}
