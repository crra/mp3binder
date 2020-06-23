package mp3binder

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bogem/id3v2"
	"github.com/dmulholl/mp3lib"
)

const (
	defaultTrackNumber           = "1"
	TagCover                     = "APIC"
	TagTrack                     = "TRCK"
	coverType                    = "Front cover"
	emptyInfoXingFrameSize int64 = 209
)

type Cover struct {
	MimeType string
	Reader   io.Reader
	Force    bool
}

type ProgressCallbackFn func(index int)

func Bind(
	out io.WriteSeeker,
	tagTemplateReader io.Reader,
	tags map[string]string,
	cover *Cover,
	progressCallback ProgressCallbackFn,
	in ...io.Reader,
) error {
	if err := writeID3Tags(out, tagTemplateReader, tags, cover); err != nil {
		return err
	}

	var (
		bitrates    = make(map[int]struct{})
		framesCount uint32
		bytesCount  uint32
	)

	if _, err := out.Write(make([]byte, emptyInfoXingFrameSize)); err != nil {
		return fmt.Errorf("can not write, %v", err)
	}

	for fileIndex, reader := range in {
		if progressCallback != nil {
			progressCallback(fileIndex)
		}

		for i := 0; true; i++ {
			frame := mp3lib.NextFrame(reader)
			if frame == nil {
				break
			}

			if i == 0 && (mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame)) {
				continue
			}

			bitrates[frame.BitRate] = struct{}{}

			if _, err := out.Write(frame.RawBytes); err != nil {
				return fmt.Errorf("can not write, %v", err)
			}

			framesCount++

			bytesCount += uint32(len(frame.RawBytes))
		}
	}

	if err := writeBitrateHeader(out, framesCount, bytesCount, len(bitrates) > 1); err != nil {
		return err
	}

	return nil
}

func writeBitrateHeader(out io.WriteSeeker, framesCount, bytesCount uint32, multipleBitrates bool) error {
	var emptyInfoXingFrameOffset int64 = int64(bytesCount) + emptyInfoXingFrameSize
	if _, err := out.Seek(emptyInfoXingFrameOffset*-1, io.SeekCurrent); err != nil {
		return fmt.Errorf("can not seek to info/xing frame, %v", err)
	}

	header := mp3lib.NewXingHeader(framesCount, bytesCount)

	if multipleBitrates {
		offset := 4 + getSideInfoSize(header)
		copy(header.RawBytes[offset:offset+4], `Info`)
	}

	if _, err := out.Write(header.RawBytes); err != nil {
		return fmt.Errorf("can not write xing/info header, %v", err)
	}

	if _, err := out.Seek(0, io.SeekCurrent); err != nil {
		return fmt.Errorf("can not seek to end of file, %v", err)
	}

	return nil
}

func getSideInfoSize(frame *mp3lib.MP3Frame) int {
	var size int

	if frame.MPEGLayer == mp3lib.MPEGLayerIII {
		if frame.MPEGVersion == mp3lib.MPEGVersion1 {
			if frame.ChannelMode == mp3lib.Mono {
				size = 17
			} else {
				size = 32
			}
		} else {
			if frame.ChannelMode == mp3lib.Mono {
				size = 9
			} else {
				size = 17
			}
		}
	}

	return size
}

func writeID3Tags(out io.Writer, tagTemplateReader io.Reader, tags map[string]string, cover *Cover) error {
	var (
		tag    = id3v2.NewEmptyTag()
		frames = make(map[string]id3v2.Framer)
	)

	if tagTemplateReader != nil {
		var (
			tagFromTemplate *id3v2.Tag
			err             error
		)

		if tagFromTemplate, err = id3v2.ParseReader(tagTemplateReader, id3v2.Options{Parse: true}); err != nil {
			return err
		}

		for id := range tagFromTemplate.AllFrames() {
			frames[id] = tagFromTemplate.GetLastFrame(id)
		}

		frames[TagTrack] = &id3v2.TextFrame{Encoding: tagFromTemplate.DefaultEncoding(), Text: defaultTrackNumber}
	}

	for id, value := range tags {
		frames[id] = &id3v2.TextFrame{Encoding: tag.DefaultEncoding(), Text: value}
	}

	if cover != nil {
		_, ok := frames[TagCover]
		if !ok || cover.Force {
			var (
				data []byte
				err  error
			)

			if data, err = ioutil.ReadAll(cover.Reader); err != nil {
				return err
			}

			frames[TagCover] = id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    cover.MimeType,
				PictureType: id3v2.PTFrontCover,
				Description: coverType,
				Picture:     data,
			}
		}
	}

	if len(frames) == 0 {
		return nil
	}

	for id, f := range frames {
		tag.AddFrame(id, f)
	}

	if _, err := tag.WriteTo(out); err != nil {
		return err
	}

	return nil
}
