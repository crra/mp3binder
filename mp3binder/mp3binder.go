package mp3binder

import (
	"context"
	"fmt"
	"io"

	"github.com/bogem/id3v2"
	"github.com/dmulholl/mp3lib"
)

const (
	emptyInfoXingFrameSize int64 = 209
)

type job struct {
	context       context.Context
	output        io.WriteSeeker
	input         []io.ReadSeeker
	tag           *id3v2.Tag
	stageObserver stageObserver
	bindObserver  bindObserver
	tagObserver   tagObserver
}

type namedJobProcessor struct {
	name      string
	processor jobProcessor
}

type (
	stageObserver func(string, string)
	bindObserver  func(int)
	tagObserver   func(string, error)
)

func discardingStageObserver(string, string) {}
func discardingBindObserver(int)             {}
func discardingTagObserver(string, error)    {}

func Bind(parent context.Context, output io.WriteSeeker, input []io.ReadSeeker, options ...Option) error {
	j := &job{
		context: parent,
		output:  output,
		input:   input,
		tag:     id3v2.NewEmptyTag(),

		stageObserver: discardingStageObserver,
		bindObserver:  discardingBindObserver,
		tagObserver:   discardingTagObserver,
	}

	jobProcessors := make(map[stage][]namedJobProcessor)

	options = append(options, bind)
	options = append(options, writeMetadata)

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
			j.stageObserver(s.String(), p.name)
			if err := p.processor(j); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeMetadata() (stage, string, jobProcessor) {
	return stageWriteMetadata, "writing metadata", func(j *job) error {
		if _, err := j.tag.WriteTo(j.output); err != nil {
			return err
		}

		return nil
	}
}

func bind() (stage, string, jobProcessor) {
	return stageBind, "Binding", func(j *job) error {
		if _, err := j.output.Write(make([]byte, emptyInfoXingFrameSize)); err != nil {
			return err
		}

		var bytesCount uint32
		var framesCount uint32
		bitrates := make(map[int]struct{})

		for fileIndex, reader := range j.input {
			j.bindObserver(fileIndex)

			// because intput is read more then once, the seek cursor is reset
			// to the beginning of the stream.
			if _, err := reader.Seek(0, io.SeekStart); err != nil {
				return err
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

				if _, err := j.output.Write(frame.RawBytes); err != nil {
					return err
				}

				framesCount++

				bytesCount += uint32(len(frame.RawBytes))
			}
		}

		if err := writeBitrateHeader(j.output, framesCount, bytesCount, len(bitrates) > 1); err != nil {
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
