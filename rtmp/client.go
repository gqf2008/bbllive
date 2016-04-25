package rtmp

import (
	"bytes"
	"strings"
	"time"
)

var clientHandler ClientHandler = new(DefaultClientHandler)

func HandleClientRTMP(h ClientHandler) {
	clientHandler = h
}

func Connect(url string) (s *RtmpNetStream, err error) {
	//rtmp://host:port/xxx/xxxx
	ss := strings.Split(url, "/")
	addr := ss[0] + "//" + ss[2] + "/" + ss[3]

	file := strings.Join(ss[4:], "/")

	conn := newNetConnection()

	err = conn.Connect(addr)
	if err != nil {
		return
	}
	s = newNetStream(conn, nil, clientHandler)
	s.play(file, "live")
	return
}
func newNetConnection() (c *RtmpNetConnection) {
	c = new(RtmpNetConnection)
	c.readChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.writeChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.createTime = time.Now()
	c.bandwidth = 512 << 10
	c.rchunks = make(map[uint32]*RtmpChunk)
	c.wchunks = make(map[uint32]*RtmpChunk)
	c.buffer = bytes.NewBuffer(nil)
	c.w_buffer = bytes.NewBuffer(nil)
	c.nextStreamId = gen_next_stream_id
	c.objectEncoding = 0
	return
}
