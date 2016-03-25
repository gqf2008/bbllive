package rtmp

import (
	"babylon/util"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

var ErrorChunkLength = errors.New("error chunk length")
var ErrorChunkType = errors.New("error chunk type")

type NetConnection interface {
	Connect(command string, args ...Args) error
	Call(command string, args ...Args) error
	Connected() bool
	Close()
	URL() string
}

func newconn(conn net.Conn, srv *Server) (c *RtmpNetConnection) {
	c = new(RtmpNetConnection)
	c.remoteAddr = conn.RemoteAddr().String()
	c.server = srv
	c.readChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.writeChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.createTime = time.Now()
	c.bandwidth = 512 << 10
	c.conn = conn
	c.br = bufio.NewReader(conn)
	c.bw = bufio.NewWriter(conn)
	c.buf = bufio.NewReadWriter(c.br, c.bw)
	c.rchunks = make(map[uint32]*RtmpChunk)
	c.wchunks = make(map[uint32]*RtmpChunk)
	//c.csid_chunk = make(map[uint32]uint32)
	// c.lastReadHeaders = make(map[uint32]*RtmpHeader)
	// c.lastWriteHeaders = make(map[uint32]*RtmpHeader)
	// c.incompletePackets = make(map[uint32]*bytes.Buffer)
	c.w_buffer = bytes.NewBuffer(nil)
	c.buffer = bytes.NewBuffer(nil)
	c.nextStreamId = gen_next_stream_id
	c.objectEncoding = 0
	return
}

type RtmpNetConnection struct {
	remoteAddr       string
	url              string
	app              string
	createTime       time.Time
	readChunkSize    int
	writeChunkSize   int
	readTimeout      time.Duration
	writeTimeout     time.Duration
	bandwidth        uint32
	limitType        byte
	wirtesequencenum uint32
	sequencenum      uint32
	totalreadbytes   uint32
	totalwritebytes  uint32
	server           *Server           // the Server on which the connection arrived
	conn             net.Conn          // i/o connection
	buf              *bufio.ReadWriter // buffered(lr,rwc), reading from bufio->limitReader->sr->rwc
	br               *bufio.Reader
	bw               *bufio.Writer
	lock             sync.Mutex // guards the following
	// incompletePackets map[uint32]*bytes.Buffer
	// lastReadHeaders   map[uint32]*RtmpHeader
	// lastWriteHeaders  map[uint32]*RtmpHeader
	rchunks        map[uint32]*RtmpChunk
	wchunks        map[uint32]*RtmpChunk
	connected      bool
	nextStreamId   func(chunkid uint32) uint32
	streamid       uint32
	objectEncoding int
	w_buffer       *bytes.Buffer
	buffer         *bytes.Buffer
	//m_timestamp    uint32
	//mchunkid       uint32
}

func (c *RtmpNetConnection) Connect(URL string, args ...Args) error {
	//rtmp://host:port/app
	log.Debug(URL)
	p := strings.Split(URL, "/")
	address := p[2]
	log.Debug(address)
	app := p[3]
	log.Debug(app)
	host := strings.Split(address, ":")
	if len(host) == 1 {
		address += ":1935"
	}
	log.Debug(address)
	conn, err := net.DialTimeout("tcp", address, time.Second*30)
	if err != nil {
		return err
	}
	c.conn = conn
	c.app = app
	c.remoteAddr = conn.RemoteAddr().String()
	c.br = bufio.NewReader(conn)
	c.bw = bufio.NewWriter(conn)
	c.buf = bufio.NewReadWriter(c.br, c.bw)
	if !client_simple_handshake(c.buf) {
		return errors.New("Client Handshake Fail")
	}
	err = sendConnect(c, app, "", "", URL)
	if err != nil {
		return err
	}
	for {
		msg, err := readMessage(c)
		if err != nil {
			c.Close()
			return err
		}
		log.Debug(msg)
		if _, ok := msg.(*ReplyMessage); ok {
			dec := newDecoder(msg.Body())
			reply := new(ReplyConnectMessage)
			reply.RtmpHeader = msg.Header()
			reply.Command = readString(dec)
			reply.TransactionId = readNumber(dec)
			//dec.readNull()
			reply.Properties = readObject(dec)
			reply.Infomation = readObject(dec)
			log.Debug(reply)
			if NetConnection_Connect_Success == getString(reply.Infomation, "code") {
				c.connected = true
				return nil
			} else {
				return errors.New(getString(reply.Infomation, "code"))
			}
		}
	}
	return nil
}

func getString(obj interface{}, key string) string {
	return obj.(Map)[key].(string)
}

func (c *RtmpNetConnection) Call(command string, args ...Args) error {
	return nil
}
func (c *RtmpNetConnection) Connected() bool {
	return c.connected
}
func (c *RtmpNetConnection) Close() {
	if c.conn == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.conn.Close()
	c.connected = false
}
func (c *RtmpNetConnection) URL() string {
	return c.url
}

func writeMessage(p *RtmpNetConnection, msg RtmpMessage) (err error) {
	if p.wirtesequencenum > p.bandwidth {
		p.totalwritebytes += p.wirtesequencenum
		p.wirtesequencenum = 0
		sendAck(p, p.totalwritebytes)
		sendPing(p)
	}
	log.Debug(p.remoteAddr, " >>>>> ", msg)
	chunk, reset, err := encodeChunk12(msg.Header(), msg.Body(), p.writeChunkSize)
	if err != nil {
		return
	}
	_, err = p.bw.Write(chunk)
	if err != nil {
		return
	}
	err = p.bw.Flush()
	if err != nil {
		return
	}
	p.wirtesequencenum += uint32(len(chunk))
	//log.Debug(">>>>> chunk ", len(chunk), " reset ", len(reset))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(msg.Header(), reset, p.writeChunkSize)
		if err != nil {
			return
		}
		_, err = p.bw.Write(chunk)
		if err != nil {
			return
		}
		err = p.bw.Flush()
		if err != nil {
			return
		}
		p.wirtesequencenum += uint32(len(chunk))
		//log.Debug(">>>>> chunk ", len(chunk), " reset ", len(reset))
	}

	return
}
func readMessage(conn *RtmpNetConnection) (msg RtmpMessage, err error) {
	if conn.sequencenum >= conn.bandwidth {
		conn.totalreadbytes += conn.sequencenum
		conn.sequencenum = 0
		sendAck(conn, conn.totalreadbytes)
	}
	msg, err = readMessage0(conn)
	if err != nil {
		return nil, err
	}
	switch msg.Header().MessageType {
	case RTMP_MSG_CHUNK_SIZE:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*ChunkSizeMessage)
		conn.readChunkSize = int(m.ChunkSize)
		return readMessage(conn)
	case RTMP_MSG_ABORT:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*AbortMessage)
		delete(conn.rchunks, m.ChunkId)
		return readMessage(conn)
	case RTMP_MSG_ACK:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		return readMessage(conn)
	case RTMP_MSG_USER:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		if _, ok := msg.(*PingMessage); ok {
			sendPong(conn)
		}
		return readMessage(conn)
	case RTMP_MSG_ACK_SIZE:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*AckWinSizeMessage)
		conn.bandwidth = m.AckWinsize
		return readMessage(conn)
	case RTMP_MSG_BANDWIDTH:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*SetPeerBandwidthMessage)
		conn.bandwidth = m.AckWinsize
		//sendAckWinsize(conn, m.AckWinsize)
		return readMessage(conn)
	case RTMP_MSG_EDGE:
		log.Debug(conn.remoteAddr, " <<<<< ", msg)
		return readMessage(conn)
	}
	return
}

type RtmpChunk struct {
	chunkid      uint32
	timestamp    uint32
	length       uint32
	mtype        byte
	streamid     uint32
	exttimestamp bool
	body         *bytes.Buffer
}

func (c *RtmpChunk) String() string {
	if c == nil {
		return "<nil>"
	}
	return fmt.Sprintf("chunkid:%v timestamp:%v msg_length:%v msg_type:%v stream_id:%v ext:%v body:%v", c.chunkid, c.timestamp, c.length, c.mtype, c.streamid, c.exttimestamp, c.body.Len())
}

func readMessage0(p *RtmpNetConnection) (msg RtmpMessage, err error) {
	tmp := p.buffer
	chunkhead, err := p.br.ReadByte()
	p.sequencenum += 1
	if err != nil {
		return nil, err
	}
	csid := uint32((chunkhead & 0x3f)) //块ID
	fmt := (chunkhead & 0xc0) >> 6     //块类型
	switch csid {
	case 0:
		u8, err := p.br.ReadByte()
		p.sequencenum += 1
		if err != nil {
			return nil, err
		}
		csid = 64 + uint32(u8)
	case 1:
		tmp.Reset()
		if _, err = io.CopyN(tmp, p.br, 2); err != nil {
			return
		}
		p.sequencenum += 2
		u16 := tmp.Bytes()
		csid = 64 + uint32(u16[0]) + 256*uint32(u16[1])
	}
	var chunk *RtmpChunk
	var exist bool
	if chunk, exist = p.rchunks[csid]; !exist {
		if fmt == 0 {
			chunk = &RtmpChunk{csid, 0, 0, 0, 0, false, bytes.NewBuffer(nil)}
			p.rchunks[csid] = chunk
		} else {
			return nil, ErrorChunkType
		}
	}

	switch fmt {
	case 0: //11字节头
		tmp.Reset()
		if _, err = io.CopyN(tmp, p.br, 11); err != nil {
			return
		}
		p.sequencenum += 11
		buf := tmp.Bytes()
		chunk.timestamp = util.BigEndian.Uint24(buf[0:3]) //type=0的时间戳为绝对时间，其他的都为相对前一个chunk的时间
		chunk.length = util.BigEndian.Uint24(buf[3:6])
		chunk.mtype = buf[6]
		chunk.streamid = util.LittleEndian.Uint32(buf[7:11])
		if chunk.timestamp == 0xffffff {
			tmp.Reset()
			if _, err = io.CopyN(tmp, p.br, 4); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			chunk.exttimestamp = true
			chunk.timestamp = util.BigEndian.Uint32(tmp.Bytes())
		}
		log.Debug("type0", chunk.timestamp, chunk)

	case 1: //7字节头

		tmp.Reset()
		if _, err = io.CopyN(tmp, p.br, 7); err != nil {
			return
		}
		p.sequencenum += 7
		buf := tmp.Bytes()
		delta := util.BigEndian.Uint24(buf[0:3])
		chunk.length = util.BigEndian.Uint24(buf[3:6])
		chunk.mtype = buf[6]
		if delta == 0xffffff {
			tmp.Reset()
			if _, err := io.CopyN(tmp, p.br, 4); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			chunk.exttimestamp = true
			delta = util.BigEndian.Uint32(tmp.Bytes())
		}
		chunk.timestamp += delta
		log.Debug("type1", delta, chunk)
	case 2: //3字节头

		tmp.Reset()
		if _, err = io.CopyN(tmp, p.br, 3); err != nil {
			return
		}
		p.sequencenum += 3
		buf := tmp.Bytes()
		delta := util.BigEndian.Uint24(buf[0:3])
		if delta == 0xffffff {
			tmp.Reset()
			if _, err := io.CopyN(tmp, p.br, 4); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			chunk.exttimestamp = true
			delta = util.BigEndian.Uint32(tmp.Bytes())
		}
		chunk.timestamp += delta
		log.Debug("type2", delta, chunk)
	case 3: //0字节头

		log.Debug("type3", chunk)
	}
	if chunk.length > 0xffffff {
		return nil, ErrorChunkLength
	}
	nRead := chunk.body.Len()
	size := int(chunk.length) - nRead
	if size > p.readChunkSize {
		size = p.readChunkSize
	}
	i, err := io.CopyN(chunk.body, p.br, int64(size))
	if err != nil {
		return
	}
	p.sequencenum += uint32(i)
	if chunk.body.Len() == int(chunk.length) {
		log.Debug("chunk", chunk)
		msg = decodeRtmpMessage1(chunk)
		// delete(p.rchunks, csid)
		chunk.body.Reset()
		return
	}
	return readMessage0(p)
}

func sendChunkSize(conn *RtmpNetConnection, size uint32) error {
	msg := new(ChunkSizeMessage)
	msg.ChunkSize = size
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_CHUNK_SIZE, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}
func sendAck(conn *RtmpNetConnection, num uint32) error {
	msg := new(AckMessage)
	msg.SequenceNumber = num
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_ACK, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}
func sendAckWinsize(conn *RtmpNetConnection, size uint32) error {
	msg := new(AckWinSizeMessage)
	msg.AckWinsize = size
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_ACK_SIZE, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPeerBandwidth(conn *RtmpNetConnection, size uint32) error {
	msg := new(SetPeerBandwidthMessage)
	msg.AckWinsize = size
	msg.LimitType = byte(2)
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_BANDWIDTH, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendStreamBegin(conn *RtmpNetConnection) error {
	msg := new(StreamBeginMessage)
	msg.EventType = RTMP_USER_STREAM_BEGIN
	msg.StreamId = conn.streamid
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendStreamRecorded(conn *RtmpNetConnection) error {
	msg := new(RecordedMessage)
	msg.EventType = RTMP_USER_RECORDED
	msg.StreamId = conn.streamid
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPing(conn *RtmpNetConnection) error {
	msg := new(PingMessage)
	msg.EventType = RTMP_USER_PING
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPong(conn *RtmpNetConnection) error {
	msg := new(PongMessage)
	msg.EventType = RTMP_USER_PONG
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendSetBufferMessage(conn *RtmpNetConnection) error {
	msg := new(SetBufferMessage)
	msg.EventType = RTMP_USER_SET_BUFLEN
	msg.StreamId = conn.streamid
	msg.Millisecond = 100
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendConnect(conn *RtmpNetConnection, app, pageUrl, swfUrl, tcUrl string) error {
	result := new(ConnectMessage)
	result.Command = "connect"
	result.TransactionId = 1
	obj := newMap()
	obj["app"] = app
	obj["audioCodecs"] = 3575
	obj["capabilities"] = 239
	obj["flashVer"] = "MAC 11,7,700,203"
	obj["fpad"] = false
	obj["objectEncoding"] = 0
	obj["pageUrl"] = pageUrl
	obj["swfUrl"] = swfUrl
	obj["tcUrl"] = tcUrl
	obj["videoCodecs"] = 252
	obj["videoFunction"] = 1
	result.Object = obj
	info := newMap()
	result.Optional = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendCreateStream(conn *RtmpNetConnection) error {
	m := new(CreateStreamMessage)
	m.Command = "createStream"
	m.TransactionId = 1
	m.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(m.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	m.RtmpHeader = head
	return writeMessage(conn, m)
}

func sendPlay(conn *RtmpNetConnection, name string, start, duration int, rest bool) error {
	m := new(PlayMessage)
	m.Command = "play"
	m.TransactionId = 1
	m.StreamName = name
	m.Start = uint64(start)
	m.Duration = uint64(duration)
	m.Rest = rest
	m.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(m.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	m.RtmpHeader = head
	return writeMessage(conn, m)
}

func sendConnectResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyConnectMessage)
	result.Command = NetStatus_Result
	result.TransactionId = 1
	pro := newMap()
	pro["fmsVer"] = SERVER_NAME + "/" + VERSION
	pro["capabilities"] = 31
	pro["mode"] = 1
	pro["Author"] = "G.Q.F/gao.qingfeng@gmail.com"
	result.Properties = pro
	info := newMap()
	info["level"] = level
	info["code"] = NetConnection_Connect_Success
	info["objectEncoding"] = uint64(conn.objectEncoding)
	result.Infomation = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendConnectSuccess(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Status, NetConnection_Connect_Success)
}

func sendConnectFailed(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_Failed)
}
func sendConnectRejected(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_Rejected)
}
func sendConnectInvalidApp(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_InvalidApp)
}
func sendConnectClose(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Status, NetConnection_Connect_Closed)
}
func sendConnectAppShutdown(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_AppShutdown)
}

func sendCreateStreamResult(conn *RtmpNetConnection, tid uint64) error {
	result := new(ReplyCreateStreamMessage)
	result.Command = NetStatus_Result
	result.TransactionId = tid
	result.StreamId = conn.streamid
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPlayResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyPlayMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 0
	info := newMap()
	info["level"] = level
	info["code"] = code
	//putString(info, "details", details)
	//putString(info, "description", "OK")
	info["clientid"] = 1
	result.Object = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPlayReset(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Reset)
}
func sendPlayStart(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Start)
}
func sendPlayStop(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Stop)
}
func sendPlayFailed(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Error, NetStream_Play_Failed)
}
func sendPlayNotFound(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Error, NetStream_Play_StreamNotFound)
}

func sendPublishResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyPublishMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 0
	info := newMap()
	info["level"] = level
	info["code"] = code
	info["clientid"] = 1
	result.Infomation = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPublishStart(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Publish_Start)
}
func sendPublishIdle(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Publish_Idle)
}
func sendPublishBadName(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Error, NetStream_Publish_BadName)
}
func sendUnpublishSuccess(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Unpublish_Success)
}

func sendStreamDataStart(conn *RtmpNetConnection) error {
	result := new(ReplyPlayMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 1
	info := newMap()
	info["level"] = Level_Status
	info["code"] = NetStream_Data_Start
	info["clientid"] = 1
	result.Object = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_META, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendSampleAccess(conn *RtmpNetConnection) error {
	return nil
}

func sendMetaData(conn *RtmpNetConnection, data *MediaFrame) error {
	head := newRtmpHeader(RTMP_CHANNEL_DATA, 0, data.Payload.Len(), RTMP_MSG_AMF_META, conn.streamid, 0)
	msg := new(MetadataMessage)
	msg.RtmpHeader = head
	msg.Payload = data.Bytes()
	return writeMessage(conn, msg)
}

func sendFullVideo(conn *RtmpNetConnection, video *MediaFrame) (err error) {
	//log.Info("=====Frame2 video", video)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	if conn.writeChunkSize > RTMP_MAX_CHUNK_SIZE {
		err = ChunkError
		return
	}
	chunk := &RtmpChunk{
		RTMP_CHANNEL_VIDEO,
		video.Timestamp,
		uint32(video.Payload.Len()),
		RTMP_MSG_VIDEO,
		conn.streamid,
		video.Timestamp > 0xffffff,
		bytes.NewBuffer(nil),
	}
	conn.wchunks[chunk.chunkid] = chunk
	buf := chunk.body
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + chunk.chunkid))
	buf.Write([]byte{0, 0, 0})
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	buf.WriteByte(byte(chunk.streamid))
	buf.WriteByte(byte(chunk.streamid >> 8))
	buf.WriteByte(byte(chunk.streamid >> 16))
	buf.WriteByte(byte(chunk.streamid >> 24))
	size := conn.writeChunkSize
	payload := video.Payload.Bytes()
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
	conn.wirtesequencenum += uint32(buf.Len())
	conn.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = buf.WriteTo(conn.conn)
	buf.Reset()
	return
}

func sendFullAudio(conn *RtmpNetConnection, audio *MediaFrame) (err error) {
	//log.Info("=====Frame2 audio", audio)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	if conn.writeChunkSize > RTMP_MAX_CHUNK_SIZE {
		err = ChunkError
		return
	}
	chunk := &RtmpChunk{
		RTMP_CHANNEL_AUDIO,
		audio.Timestamp,
		uint32(audio.Payload.Len()),
		RTMP_MSG_AUDIO,
		conn.streamid,
		audio.Timestamp > 0xffffff,
		bytes.NewBuffer(nil),
	}
	conn.wchunks[chunk.chunkid] = chunk
	buf := chunk.body
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + chunk.chunkid))
	buf.Write([]byte{0, 0, 0})
	buf.WriteByte(byte(chunk.length >> 16))
	buf.WriteByte(byte(chunk.length >> 8))
	buf.WriteByte(byte(chunk.length))
	buf.WriteByte(chunk.mtype)
	buf.WriteByte(byte(chunk.streamid))
	buf.WriteByte(byte(chunk.streamid >> 8))
	buf.WriteByte(byte(chunk.streamid >> 16))
	buf.WriteByte(byte(chunk.streamid >> 24))
	size := conn.writeChunkSize
	payload := audio.Payload.Bytes()
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
	conn.wirtesequencenum += uint32(buf.Len())
	conn.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = buf.WriteTo(conn.conn)
	buf.Reset()
	return
}

func sendVideo(conn *RtmpNetConnection, video *MediaFrame) (err error) {
	chunk, exist := conn.wchunks[RTMP_CHANNEL_VIDEO]
	if !exist {
		return sendFullVideo(conn, video)
	}
	buf := chunk.body
	chunk.length = uint32(video.Payload.Len())
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_8 + chunk.chunkid))
	delta := video.Timestamp - chunk.timestamp
	//log.Info("video timestamp", video.Timestamp, chunk.timestamp, delta)
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
	size := conn.writeChunkSize
	payload := video.Payload.Bytes()
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
	_, err = buf.WriteTo(conn.w_buffer)
	buf.Reset()
	return

}

func sendAudio(conn *RtmpNetConnection, audio *MediaFrame) (err error) {
	chunk, exist := conn.wchunks[RTMP_CHANNEL_AUDIO]
	if !exist {
		return sendFullAudio(conn, audio)
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
	size := conn.writeChunkSize
	payload := audio.Payload.Bytes()
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
	_, err = buf.WriteTo(conn.w_buffer)
	buf.Reset()
	return

}

func flush(conn *RtmpNetConnection) (err error) {
	if conn.writeChunkSize > RTMP_MAX_CHUNK_SIZE {
		err = ChunkError
		return
	}
	if conn.w_buffer.Len() == 0 {
		return
	}
	conn.wirtesequencenum += uint32(conn.w_buffer.Len())
	conn.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err = conn.w_buffer.WriteTo(conn.conn)
	conn.w_buffer.Reset()
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	return
}
