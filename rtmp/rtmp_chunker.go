package rtmp

import (
	"bytes"
)

type RtmpChunker struct {
	writeChunkSize int
	wchunks        map[uint32]*RtmpChunk
	streamid       uint32
	w_buffer       *bytes.Buffer
}

func NewRtmpChunker(streamid uint32) *RtmpChunker {
	return &RtmpChunker{RTMP_DEFAULT_CHUNK_SIZE, make(map[uint32]*RtmpChunk, 0), streamid, bytes.NewBuffer(nil)}
}

func (r *RtmpChunker) Reset() {
	r.w_buffer.Reset()
}
func (r *RtmpChunker) Bytes() []byte {
	return r.w_buffer.Bytes()
}

func (r *RtmpChunker) writeMetadata(data *MediaFrame) error {
	chunk := &RtmpChunk{
		RTMP_CHANNEL_DATA,
		0,
		0,
		uint32(data.Payload.Len()),
		RTMP_MSG_AMF_META,
		r.streamid,
		false,
		bytes.NewBuffer(nil),
	}
	r.wchunks[chunk.chunkid] = chunk
	buf := chunk.body
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + chunk.chunkid))
	buf.Write([]byte{0, 0, 0})
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(RTMP_MSG_AMF_META)
	buf.WriteByte(byte(chunk.streamid))
	buf.WriteByte(byte(chunk.streamid >> 8))
	buf.WriteByte(byte(chunk.streamid >> 16))
	buf.WriteByte(byte(chunk.streamid >> 24))
	size := r.writeChunkSize
	payload := data.Payload.Bytes()
	for {
		if len(payload) > size {
			buf.Write(payload[0:size])
			payload = payload[size:]
			if len(payload) > 0 {
				buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + chunk.chunkid))
			}
		} else {
			buf.Write(payload)
			break
		}
	}
	r.w_buffer.Write(buf.Bytes())
	//log.Info("writeMetadata", r.w_buffer.Len(), chunk, data)
	buf.Reset()
	return nil
}

func (r *RtmpChunker) writeFullVideo(video *MediaFrame) (err error) {

	chunk := &RtmpChunk{
		RTMP_CHANNEL_VIDEO,
		video.Timestamp,
		0,
		uint32(video.Payload.Len()),
		RTMP_MSG_VIDEO,
		r.streamid,
		video.Timestamp > 0xffffff,
		bytes.NewBuffer(nil),
	}
	r.wchunks[chunk.chunkid] = chunk
	buf := chunk.body
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + chunk.chunkid))
	if chunk.exttimestamp {
		buf.Write([]byte{0xff, 0xff, 0xff})
	} else {
		buf.WriteByte(byte(video.Timestamp >> 16))
		buf.WriteByte(byte(video.Timestamp >> 8))
		buf.WriteByte(byte(video.Timestamp))
	}
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	buf.WriteByte(byte(chunk.streamid))
	buf.WriteByte(byte(chunk.streamid >> 8))
	buf.WriteByte(byte(chunk.streamid >> 16))
	buf.WriteByte(byte(chunk.streamid >> 24))
	size := r.writeChunkSize
	payload := video.Payload.Bytes()
	var chunk4 bool
	for {
		if len(payload) > size {
			buf.Write(payload[0:size])
			payload = payload[size:]
			if len(payload) > 0 {
				if !chunk4 {
					buf.Write([]byte{byte(RTMP_CHUNK_HEAD_4 + chunk.chunkid), 0, 0, 0})
				} else {
					buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + chunk.chunkid))
					chunk4 = true
				}
			}
		} else {
			buf.Write(payload)
			break
		}
	}
	r.w_buffer.Write(buf.Bytes())
	//log.Info("writeFullVideo", r.w_buffer.Len(), chunk, video)
	buf.Reset()
	return
}

func (r *RtmpChunker) writeFullAudio(audio *MediaFrame) (err error) {

	chunk := &RtmpChunk{
		RTMP_CHANNEL_AUDIO,
		audio.Timestamp,
		0,
		uint32(audio.Payload.Len()),
		RTMP_MSG_AUDIO,
		r.streamid,
		audio.Timestamp > 0xffffff,
		bytes.NewBuffer(nil),
	}
	r.wchunks[chunk.chunkid] = chunk

	buf := chunk.body
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + chunk.chunkid))
	if chunk.exttimestamp {
		buf.Write([]byte{0xff, 0xff, 0xff})
	} else {
		buf.WriteByte(byte(audio.Timestamp >> 16))
		buf.WriteByte(byte(audio.Timestamp >> 8))
		buf.WriteByte(byte(audio.Timestamp))
	}
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	buf.WriteByte(byte(chunk.streamid))
	buf.WriteByte(byte(chunk.streamid >> 8))
	buf.WriteByte(byte(chunk.streamid >> 16))
	buf.WriteByte(byte(chunk.streamid >> 24))
	size := r.writeChunkSize
	payload := audio.Payload.Bytes()
	var chunk4 bool
	for {
		if len(payload) > size {
			buf.Write(payload[0:size])
			payload = payload[size:]
			if len(payload) > 0 {
				if !chunk4 {
					buf.Write([]byte{byte(RTMP_CHUNK_HEAD_4 + chunk.chunkid), 0, 0, 0})
				} else {
					buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + chunk.chunkid))
					chunk4 = true
				}
			}
		} else {
			buf.Write(payload)
			break
		}
	}
	r.w_buffer.Write(buf.Bytes())
	//log.Info("writeFullAudio", r.w_buffer.Len(), chunk, audio)
	buf.Reset()
	return
}

func (r *RtmpChunker) writeVideo(video *MediaFrame) (err error) {

	chunk, exist := r.wchunks[RTMP_CHANNEL_VIDEO]
	if !exist {
		return r.writeFullVideo(video)
	}

	buf := chunk.body
	chunk.length = uint32(video.Payload.Len())
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_8 + chunk.chunkid))
	delta := video.Timestamp - chunk.timestamp
	if delta > 0xffffff {
		buf.Write([]byte{0xff, 0xff, 0xff})
	} else {
		buf.WriteByte(byte(delta >> 16))
		buf.WriteByte(byte(delta >> 8))
		buf.WriteByte(byte(delta))
	}
	chunk.timestamp += delta
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	if delta > 0xffffff {
		buf.WriteByte(byte(delta))
		buf.WriteByte(byte(delta >> 8))
		buf.WriteByte(byte(delta >> 16))
		buf.WriteByte(byte(delta >> 24))
	}
	size := r.writeChunkSize
	payload := video.Payload.Bytes()
	var chunk4 bool
	for {
		if len(payload) > size {
			buf.Write(payload[0:size])
			payload = payload[size:]
			if len(payload) > 0 {
				if !chunk4 {
					buf.Write([]byte{byte(RTMP_CHUNK_HEAD_4 + chunk.chunkid), 0, 0, 0})
				} else {
					buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + chunk.chunkid))
					chunk4 = true
				}
			}
		} else {
			buf.Write(payload)
			break
		}
	}
	r.w_buffer.Write(buf.Bytes())
	//log.Info("writeVideo", r.w_buffer.Len(), chunk, video)
	buf.Reset()
	return

}

func (r *RtmpChunker) writeAudio(audio *MediaFrame) (err error) {

	chunk, exist := r.wchunks[RTMP_CHANNEL_AUDIO]
	if !exist {
		return r.writeFullAudio(audio)
	}

	buf := chunk.body
	chunk.length = uint32(audio.Payload.Len())
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_8 + chunk.chunkid))
	delta := audio.Timestamp - chunk.timestamp
	//log.Info("audio timestamp", audio.Timestamp, chunk.timestamp, delta)
	if delta > 0xffffff {
		buf.Write([]byte{0xff, 0xff, 0xff})
	} else {
		buf.WriteByte(byte(delta >> 16))
		buf.WriteByte(byte(delta >> 8))
		buf.WriteByte(byte(delta))
	}
	chunk.timestamp += delta
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	if delta > 0xffffff {
		buf.WriteByte(byte(delta))
		buf.WriteByte(byte(delta >> 8))
		buf.WriteByte(byte(delta >> 16))
		buf.WriteByte(byte(delta >> 24))
	}
	size := r.writeChunkSize
	payload := audio.Payload.Bytes()
	var chunk4 bool
	for {
		if len(payload) > size {
			buf.Write(payload[0:size])
			payload = payload[size:]
			if len(payload) > 0 {
				if !chunk4 {
					buf.Write([]byte{byte(RTMP_CHUNK_HEAD_4 + chunk.chunkid), 0, 0, 0})
				} else {
					buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + chunk.chunkid))
					chunk4 = true
				}
			}
		} else {
			buf.Write(payload)
			break
		}
	}
	r.w_buffer.Write(buf.Bytes())
	//log.Info("writeAudio", r.w_buffer.Len(), chunk, audio)
	buf.Reset()
	return
}
