package hls

import (
	"bbllive/rtmp"
	"bytes"
	"errors"
	"io"
)

var (
	PMT_PID   uint16 = 222
	PCR_PID   uint16 = 222
	VIDEO_PID uint16 = 222
	AUDIO_PID uint16 = 222
)

func NewTsWriter(sps, pps, adts []byte) *tsWriter {
	return &tsWriter{
		sps:       sps,
		pps:       pps,
		adts:      adts,
		pmt_pid:   PMT_PID,
		pcr_pid:   PCR_PID,
		video_pid: VIDEO_PID,
		audio_pid: AUDIO_PID,
		buffer:    bytes.NewBuffer(nil),
	}.makePatPmt()
}

var ErrFrameType = errors.New("unsupported frame type")

type tsWriter struct {
	sps       []byte
	pps       []byte
	adts      []byte
	pat       []byte
	pmt       []byte
	pmt_pid   uint16
	pcr_pid   uint16
	video_pid uint16
	audio_pid uint16
	buffer    *bytes.Buffer
	duration  uint64
}

func (w *tsWriter) makePatPmt() *tsWriter {
	return w
}

func (w *tsWriter) SetSPS(sps []byte) {
	w.sps = sps
}

func (w *tsWriter) SetPPS(pps []byte) {
	w.pps = pps
}

func (w *tsWriter) SetADTS(adts []byte) {
	w.adts = adts
}

func (w *tsWriter) Write(frame *rtmp.MediaFrame) error {
	if w.buffer.Len() == 0 {
		w.buffer.Write(w.pat)
		w.buffer.Write(w.pmt)
	}
	if frame.Type == rtmp.RTMP_MSG_VIDEO {
		w.buffer.Write(unpack_ts(w.video_pid, unpack_avc_pes(w.sps, w.pps, frame)))
		return nil
	}
	if frame.Type == rtmp.RTMP_MSG_AUDIO {
		w.buffer.Write(unpack_ts(w.video_pid, unpack_aac_pes(w.adts, frame)))
		return nil
	}
	return ErrFrameType
}

func (w *tsWriter) WriteTo(o io.Writer) (n int64, err error) {
	return w.buffer.WriteTo(o)
}

func (w *tsWriter) Bytes() []byte {
	return w.buffer.Bytes()
}

func (w *tsWriter) Reset() {
	w.buffer.Reset()
}

func (w *tsWriter) Duration() uint64 {
	return w.duration
}

func unpack_avc_pes(sps, pps []byte, frame *rtmp.MediaFrame) []byte {
	return nil
}

func unpack_aac_pes(adts []byte, frame *rtmp.MediaFrame) []byte {
	return nil
}

func unpack_ts(pid uint16, pes []byte) []byte {
	return nil
}
