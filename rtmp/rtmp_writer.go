package rtmp

import (
	"bytes"
	"io"
)

func NewRtmpWriter(chunkSize int, streamid uint32) *rtmpWriter {
	return &rtmpWriter{
		avconfig:  []byte{},
		chunkSize: chunkSize,
		streamid:  streamid,
		wchunk:    make(map[uint32]*RtmpChunk, 0),
	}
}

type rtmpWriter struct {
	avconfig  []byte
	chunkSize int
	streamid  uint32
	wchunk    map[uint32]*RtmpChunk
	buffer    *bytes.Buffer
}

func (w *rtmpWriter) SetMetadata(frame *MediaFrame) error {
	return nil
}
func (w *rtmpWriter) SetVideoConfig(frame *MediaFrame) error {
	return nil
}
func (w *rtmpWriter) SetAudioConfig(frame *MediaFrame) error {
	return nil
}

func (w *rtmpWriter) Write(frame *MediaFrame) error {
	return nil
}

func (w *rtmpWriter) WriteTo(o io.Writer) (n int64, err error) {
	return
}

func (w *rtmpWriter) Bytes() []byte {
	return w.buffer.Bytes()
}

func (w *rtmpWriter) ResetWChunk() {
	w.wchunk = make(map[uint32]*RtmpChunk, 0)
}

func (w *rtmpWriter) ResetBuffer() {
	w.buffer.Reset()
}

func (w *rtmpWriter) Reset() {
	w.wchunk = make(map[uint32]*RtmpChunk, 0)
	w.buffer.Reset()
}
