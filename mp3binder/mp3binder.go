package mp3binder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/dmulholl/mp3lib"
)

var ErrUnusableOption = errors.New("unusable option")

const (
	emptyInfoXingFrameSize int64 = 209
	tagTitle                     = "TIT2"
)

type job struct {
	context   context.Context
	output    io.WriteSeeker
	audioOnly io.ReadWriteSeeker
	inputs    []io.ReadSeeker

	tagResolver tagResolver
	tag         *id3v2.Tag
	metadata    []*id3v2.Tag

	inputDurations  []time.Duration
	stageObserver   stageObserver
	bindObserver    bindObserver
	tagCopyObserver tagCopyObserver
	tagObserver     tagObserver
}

type namedJobProcessor struct {
	name      string
	processor jobProcessor
}

type (
	stageObserver   func(string, string)
	bindObserver    func(int)
	tagCopyObserver func(string, string, error)
	tagObserver     func(string, string, error)
)

type tagResolver interface {
	DescriptionFor(string) (string, error)
}

func discardingStageObserver(string, string)          {}
func discardingBindObserver(int)                      {}
func discardingTagObserver(string, string, error)     {}
func discardingTagCopyObserver(string, string, error) {}

type binder struct {
	tagResolver tagResolver
}

func New(tagResolver tagResolver) *binder {
	return &binder{
		tagResolver: tagResolver,
	}
}

func (b *binder) Bind(parent context.Context, output io.WriteSeeker, audioOnly io.ReadWriteSeeker, input []io.ReadSeeker, o ...any) error {
	options := make([]Option, len(o))

	for i, op := range o {
		option, ok := op.(Option)
		if !ok {
			return ErrUnusableOption
		}

		options[i] = option
	}

	return Bind(parent, b.tagResolver, output, audioOnly, input, options...)
}

func Bind(parent context.Context, tagResolver tagResolver, output io.WriteSeeker, audioOnly io.ReadWriteSeeker, input []io.ReadSeeker, options ...Option) error {
	j := &job{
		context:   parent,
		output:    output,
		audioOnly: audioOnly,
		inputs:    input,

		tagResolver: tagResolver,

		tag:            id3v2.NewEmptyTag(),
		inputDurations: make([]time.Duration, len(input)),
		metadata:       make([]*id3v2.Tag, len(input)),

		stageObserver:   discardingStageObserver,
		bindObserver:    discardingBindObserver,
		tagObserver:     discardingTagObserver,
		tagCopyObserver: discardingTagCopyObserver,
	}

	jobProcessors := make(map[stage][]namedJobProcessor)

	options = append(options, bind, writeMetadata, combineId3AndAudio)

	for _, o := range options {
		stage, name, processor := o()
		jobProcessors[stage] = append(jobProcessors[stage], namedJobProcessor{
			name:      name,
			processor: processor,
		})
	}

	// process all stages
	for s := stage(0); s < stageLastElement; s++ {
		for _, p := range jobProcessors[s] {
			if s != stageInit {
				j.stageObserver(s.String(), p.name)
			}
			if err := p.processor(j); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeMetadata() (stage, string, jobProcessor) {
	return stageWriteMetadata, "writing metadata", func(j *job) error {
		var start time.Duration

		for i, m := range j.metadata {
			title := fmt.Sprintf("Chapter: %d", i+1)
			intputTitle := m.GetTextFrame(tagTitle).Text
			if intputTitle != "" {
				title = fmt.Sprintf("%s: %s", title, intputTitle)
			}

			end := start + j.inputDurations[i]
			j.tagObserver(fmt.Sprintf("Chapter: %d from '%s' to '%s'", i+1, start.Round(time.Second), end.Round(time.Second)), title, nil)

			j.tag.AddChapterFrame(id3v2.ChapterFrame{
				ElementID: fmt.Sprintf("c%d", i),
				StartTime: start,
				EndTime:   end,
				Title: &id3v2.TextFrame{
					Encoding: id3v2.EncodingUTF8,
					Text:     title,
				},
			})

			start = end
		}

		if _, err := j.tag.WriteTo(j.output); err != nil {
			return err
		}

		return nil
	}
}

func bind() (stage, string, jobProcessor) {
	return stageBind, "Binding", func(j *job) error {
		if _, err := j.audioOnly.Write(make([]byte, emptyInfoXingFrameSize)); err != nil {
			return err
		}

		var bytesCount uint32
		var framesCount uint32
		var lastBitrate int
		var multipleBitrates bool

		for fileIndex, reader := range j.inputs {
			_ = lastBitrate // linter: if there are no frames in the file, this value will never set
			j.bindObserver(fileIndex)

			// because intput could be read more then once, the seek cursor is reset
			// to the beginning of the stream.
			if _, err := reader.Seek(0, io.SeekStart); err != nil {
				return err
			}

			if j.metadata[fileIndex] == nil {
				j.metadata[fileIndex] = id3v2.NewEmptyTag()
			}

			for i := 0; true; i++ {
				obj := mp3lib.NextObject(reader)
				if obj == nil {
					break
				}

				switch obj := obj.(type) {
				case *mp3lib.MP3Frame:
					if i == 0 && (mp3lib.IsXingHeader(obj) || mp3lib.IsVbriHeader(obj)) {
						continue
					}

					if lastBitrate == 0 {
						lastBitrate = obj.BitRate
					}

					if !multipleBitrates && lastBitrate != obj.BitRate {
						multipleBitrates = true
					}

					if _, err := j.audioOnly.Write(obj.RawBytes); err != nil {
						return err
					}

					j.inputDurations[fileIndex] += duration(obj)

					framesCount++

					bytesCount += uint32(len(obj.RawBytes))

				case *mp3lib.ID3v2Tag:
					tag, err := id3v2.ParseReader(bytes.NewReader(obj.RawBytes), id3v2.Options{Parse: true})
					if err != nil {
						return err
					}

					for id := range tag.AllFrames() {
						j.metadata[fileIndex].AddFrame(id, tag.GetLastFrame(id))
					}

				default:
					continue
				}
			}
		}

		if err := writeBitrateHeader(j.audioOnly, framesCount, bytesCount, multipleBitrates); err != nil {
			return err
		}

		return nil
	}
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

func duration(frame *mp3lib.MP3Frame) time.Duration {
	return time.Duration(int64(float64(time.Millisecond) * (1000 / float64(frame.SamplingRate)) * float64(frame.SampleCount)))
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

func combineId3AndAudio() (stage, string, jobProcessor) {
	return stageCombineId3AndAudio, "combining id3 and audio", func(j *job) error {
		if _, err := j.audioOnly.Seek(0, io.SeekStart); err != nil {
			return err
		}

		if _, err := io.Copy(j.output, j.audioOnly); err != nil {
			return err
		}

		return nil
	}
}
