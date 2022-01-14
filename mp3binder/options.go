//go:generate stringer -type=stage -trimprefix stage
package mp3binder

import (
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
	// 		j.tag.AddFrame(tagIdTrack, &id3v2.TextFrame{Encoding: j.tag.DefaultEncoding(), Text: defaultTrackNumber})

)

type (
	jobProcessor func(*job) error
	Option       func() (stage, string, jobProcessor)
)

func ActionObserver(f stageObserver) Option {
	return func() (stage, string, jobProcessor) {
		return stageInit, "Stage observer", func(j *job) error {
			j.stageObserver = f

			return nil
		}
	}
}

func BindObserver(f bindObserver) Option {
	return func() (stage, string, jobProcessor) {
		return stageInit, "Bind observer", func(j *job) error {
			j.bindObserver = f

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
				j.tag.AddFrame(id, tagFromTemplate.GetLastFrame(id))
			}

			return nil
		}
	}
}

func ApplyMetadata(tags map[string]string) Option {
	return func() (stage, string, jobProcessor) {
		return stageBeforeWriteMetadata, "applying metadata", func(j *job) error {
			for id, value := range tags {
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
