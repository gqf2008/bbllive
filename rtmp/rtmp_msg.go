package rtmp

import (
	"bbllive/util"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ChunkError = errors.New("error chunk size")
	//0100,0011
	videoframetype = map[byte]string{
		1: "keyframe (for AVC, a seekable frame)",
		2: "inter frame (for AVC, a non-seekable frame)",
		3: "disposable inter frame (H.263 only)",
		4: "generated keyframe (reserved for server use only)",
		5: "video info/command frame"}
	videocodec = map[byte]string{
		1: "JPEG (currently unused)",
		2: "Sorenson H.263",
		3: "Screen video",
		4: "On2 VP6",
		5: "On2 VP6 with alpha channel",
		6: "Screen video version 2",
		7: "AVC"}
	audioformat = map[byte]string{
		0:  "Linear PCM, platform endian",
		1:  "ADPCM",
		2:  "MP3",
		3:  "Linear PCM, little endian",
		4:  "Nellymoser 16kHz mono",
		5:  "Nellymoser 8kHz mono",
		6:  "Nellymoser",
		7:  "G.711 A-law logarithmic PCM",
		8:  "G.711 mu-law logarithmic PCM",
		9:  "reserved",
		10: "AAC",
		11: "Speex",
		14: "MP3 8Khz",
		15: "Device-specific sound"}

	samplerate = map[byte]string{
		0: "5.5kHz",
		1: "11kHz",
		2: "22kHz",
		3: "44kHz"}
	samplelength = map[byte]string{
		0: "8Bit",
		1: "16Bit"}
	audiotype = map[byte]string{
		0: "Mono",
		1: "Stereo"}
)

type Payload []byte

// type StreamPacket struct {
// 	Id             int
// 	Timestamp      uint32
// 	Type           byte //8 audio,9 video
// 	VideoFrameType byte //4bit
// 	VideoCodecID   byte //4bit
// 	AudioFormat    byte //4bit
// 	SamplingRate   byte //2bit
// 	SampleLength   byte //1bit
// 	AudioType      byte //1bit
// 	Payload        Payload
// }

// func (p *StreamPacket) String() string {
// 	if p.Type == RTMP_MSG_AUDIO {
// 		return fmt.Sprintf("Audio StreamPacket Timestamp/%v Type/%v AudioFromat/%v SampleRate/%v SampleLength/%v AudioType/%v Payload/%v", p.Timestamp, p.Type, audioformat[p.AudioFormat], samplerate[p.SamplingRate], samplelength[p.SampleLength], audiotype[p.AudioType], len(p.Payload))
// 	} else if p.Type == RTMP_MSG_VIDEO {
// 		return fmt.Sprintf("Video StreamPacket Timestamp/%v Type/%v VideoFrameType/%v VideoCodecID/%v Payload/%v", p.Timestamp, p.Type, videoframetype[p.VideoFrameType], videocodec[p.VideoCodecID], len(p.Payload))
// 	}
// 	return fmt.Sprintf("StreamPacket Timestamp/%v Type/%v Payload/%v", p.Timestamp, p.Type, len(p.Payload))
// }

// func (p *StreamPacket) isKeyFrame() bool {
// 	return p.VideoFrameType == 1 || p.VideoFrameType == 4
// }

type RtmpMessage interface {
	Header() *RtmpHeader
	Body() Payload
	String() string
}

type RtmpHeader struct {
	ChunkType       byte   `json:""` /*head size 2 bit*/
	ChunkId         uint32 `json:""` /* chunk stream id , 6 bit*/
	Timestamp       uint32 `json:""` /* timestamp (delta) 3 byte*/
	MessageLength   uint32 `json:""` /* message length 3 byte*/
	MessageType     byte   `json:""` /* message type id 1 byte*/
	StreamId        uint32 `json:""` /* message stream id 4 byte*/
	ExtendTimestamp uint32 `json:",omitempty"`
}

func (p *RtmpHeader) String() string {
	return fmt.Sprintf("RtmpHeader %+v", *p)
}

func (p *RtmpHeader) Clone() *RtmpHeader {
	head := new(RtmpHeader)
	head.ChunkId = p.ChunkId
	head.Timestamp = p.Timestamp
	head.MessageLength = p.MessageLength
	head.MessageType = p.MessageType
	head.StreamId = p.StreamId
	head.ExtendTimestamp = p.ExtendTimestamp
	return head
}

func newRtmpHeader(csid uint32, time uint32, length int, ttype int, streamid uint32, exttime uint32) *RtmpHeader {
	head := new(RtmpHeader)
	head.ChunkId = csid
	head.Timestamp = time
	head.MessageLength = uint32(length)
	head.MessageType = byte(ttype)
	head.StreamId = streamid
	head.ExtendTimestamp = exttime
	return head
}

type ControlMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *ControlMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ControlMessage) Body() Payload {
	return p.Payload
}
func (p *ControlMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ControlMessage(%d) %v", len(p.Payload), string(b))
}

type ChunkSizeMessage struct {
	ControlMessage
	ChunkSize uint32 //4byte
}

func (p *ChunkSizeMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ChunkSizeMessage) Body() Payload {
	return p.Payload
}
func (p *ChunkSizeMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ChunkSizeMessage(%d) %v", len(p.Payload), string(b))
}
func (p *ChunkSizeMessage) Encode() {
	p.Payload = make(Payload, 4)
	util.BigEndian.PutUint32(p.Payload, p.ChunkSize)
}

type AbortMessage struct {
	ControlMessage
	ChunkId uint32 //4byte
}

func (p *AbortMessage) Encode() {
	p.Payload = make(Payload, 4)
	util.BigEndian.PutUint32(p.Payload, p.ChunkId)
}

func (p *AbortMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *AbortMessage) Body() Payload {
	return p.Payload
}
func (p *AbortMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("AbortMessage(%d) %v", len(p.Payload), string(b))
}

type AckMessage struct {
	ControlMessage
	SequenceNumber uint32 //4byte
}

func (p *AckMessage) Encode() {
	p.Payload = make(Payload, 4)
	util.BigEndian.PutUint32(p.Payload, p.SequenceNumber)
}

func (p *AckMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *AckMessage) Body() Payload {
	return p.Payload
}
func (p *AckMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("AckMessage(%d) %v", len(p.Payload), string(b))
}

type UserControlMessage struct {
	ControlMessage
	EventType uint16
	EventData Payload
}

func (p *UserControlMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *UserControlMessage) Body() Payload {
	return p.Payload
}
func (p *UserControlMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("UserControlMessage(%d) %v", len(p.Payload), string(b))
}

type AckWinSizeMessage struct {
	ControlMessage
	AckWinsize uint32 //4byte
}

func (p *AckWinSizeMessage) Encode() {
	p.Payload = make(Payload, 4)
	util.BigEndian.PutUint32(p.Payload, p.AckWinsize)
}

func (p *AckWinSizeMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *AckWinSizeMessage) Body() Payload {
	return p.Payload
}
func (p *AckWinSizeMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("AckWinSizeMessage(%d) %v", len(p.Payload), string(b))
}

type SetPeerBandwidthMessage struct {
	ControlMessage
	AckWinsize uint32 //4byte
	LimitType  byte
}

func (p *SetPeerBandwidthMessage) Encode() {
	p.Payload = make(Payload, 5)
	util.BigEndian.PutUint32(p.Payload, p.AckWinsize)
	p.Payload[4] = p.LimitType
}

func (p *SetPeerBandwidthMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *SetPeerBandwidthMessage) Body() Payload {
	return p.Payload
}
func (p *SetPeerBandwidthMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("SetPeerBandwidthMessage(%d) %v", len(p.Payload), string(b))
}

type EdegMessage struct {
	ControlMessage
}

func (p *EdegMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *EdegMessage) Body() Payload {
	return p.Payload
}
func (p *EdegMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("EdegMessage(%d) %v", len(p.Payload), string(b))
}

type StreamBeginMessage struct {
	UserControlMessage
	StreamId uint32
}

func (p *StreamBeginMessage) Encode() {
	p.Payload = make(Payload, 6)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.StreamId)
	p.EventData = p.Payload[2:]
}

func (p *StreamBeginMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *StreamBeginMessage) Body() Payload {
	return p.Payload
}
func (p *StreamBeginMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("StreamBeginMessage(%d) %v", len(p.Payload), string(b))
}

type StreamEOFMessage struct {
	UserControlMessage
	StreamId uint32
}

func (p *StreamEOFMessage) Encode() {
	p.Payload = make(Payload, 6)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.StreamId)
	p.EventData = p.Payload[2:]
}

func (p *StreamEOFMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *StreamEOFMessage) Body() Payload {
	return p.Payload
}
func (p *StreamEOFMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("StreamEOFMessage(%d) %v", len(p.Payload), string(b))
}

type StreamDryMessage struct {
	UserControlMessage
	StreamId uint32
}

func (p *StreamDryMessage) Encode() {
	p.Payload = make(Payload, 6)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.StreamId)
	p.EventData = p.Payload[2:]
}

func (p *StreamDryMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *StreamDryMessage) Body() Payload {
	return p.Payload
}
func (p *StreamDryMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("StreamDryMessage(%d) %v", len(p.Payload), string(b))
}

type UnknownUserMessage struct {
	UserControlMessage
}

func (p *UnknownUserMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *UnknownUserMessage) Body() Payload {
	return p.Payload
}
func (p *UnknownUserMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("UnknownUserMessage(%d) %v", len(p.Payload), string(b))
}

type SetBufferMessage struct {
	UserControlMessage
	StreamId    uint32
	Millisecond uint32
}

func (p *SetBufferMessage) Encode() {
	p.Payload = make(Payload, 2+4+4)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.StreamId)
	util.BigEndian.PutUint32(p.Payload[6:], p.Millisecond)
	p.EventData = p.Payload[2:]
}

func (p *SetBufferMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *SetBufferMessage) Body() Payload {
	return p.Payload
}
func (p *SetBufferMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("SetBufferMessage(%d) %v", len(p.Payload), string(b))
}

type RecordedMessage struct {
	UserControlMessage
	StreamId uint32
}

func (p *RecordedMessage) Encode() {
	p.Payload = make(Payload, 2+4)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.StreamId)
	p.EventData = p.Payload[2:]
}

func (p *RecordedMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *RecordedMessage) Body() Payload {
	return p.Payload
}
func (p *RecordedMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("RecordedMessage(%d) %v", len(p.Payload), string(b))
}

type PingMessage struct {
	UserControlMessage
	Timestamp uint32
}

func (p *PingMessage) Encode() {
	p.Payload = make(Payload, 2+4)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	util.BigEndian.PutUint32(p.Payload[2:], p.Timestamp)
	p.EventData = p.Payload[2:]
}

func (p *PingMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *PingMessage) Body() Payload {
	return p.Payload
}
func (p *PingMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("PingMessage(%d) %v", len(p.Payload), string(b))
}

type PongMessage struct {
	UserControlMessage
}

func (p *PongMessage) Encode() {
	p.Payload = make(Payload, 2)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	p.EventData = p.Payload[2:]
}

func (p *PongMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *PongMessage) Body() Payload {
	return p.Payload
}
func (p *PongMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("PongMessage(%d) %v", len(p.Payload), string(b))
}

type BufferEndMessage struct {
	UserControlMessage
}

func (p *BufferEndMessage) Encode() {
	p.Payload = make(Payload, 2)
	util.BigEndian.PutUint16(p.Payload, p.EventType)
	p.EventData = p.Payload[2:]
}
func (p *BufferEndMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *BufferEndMessage) Body() Payload {
	return p.Payload
}
func (p *BufferEndMessage) String() string {
	b, err := json.Marshal(*p)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("BufferEndMessage(%d) %v", len(p.Payload), string(b))
}

type CommandMessage struct {
	RtmpHeader    *RtmpHeader
	Payload       Payload `json:"-"`
	Command       string
	TransactionId uint64
}

func (p *CommandMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *CommandMessage) Body() Payload {
	return p.Payload
}

func (m *CommandMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("CommandMessage(%d) %v", len(m.Payload), string(b))
}

type ConnectMessage struct {
	CommandMessage
	Object   interface{} `json:",omitempty"`
	Optional interface{} `json:",omitempty"`
}

func (p *ConnectMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	if p.Object != nil {
		amf.writeMap(p.Object.(Map))
	}
	if p.Optional != nil {
		amf.writeMap(p.Optional.(Map))
	}
	p.Payload = amf.Bytes()
}

func (p *ConnectMessage) Encode3() {
	p.Encode0()
	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.Write(p.Payload)
	p.Payload = buf.Bytes()
}

func (p *ConnectMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ConnectMessage) Body() Payload {
	return p.Payload
}

func (m *ConnectMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ConnectMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyConnectMessage struct {
	CommandMessage
	Properties interface{} `json:",omitempty"`
	Infomation interface{} `json:",omitempty"`
}

func (p *ReplyConnectMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	if p.Properties != nil {
		amf.writeMap(p.Properties.(Map))
	}
	if p.Infomation != nil {
		amf.writeMap(p.Infomation.(Map))
	}
	p.Payload = amf.Bytes()
}

//func (p *ReplyConnectMessage) Encode3() {
//	p.Encode0()
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *ReplyConnectMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyConnectMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyConnectMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyConnectMessage(%d) %v", len(m.Payload), string(b))
}

type CallMessage struct {
	CommandMessage
	Object   interface{} `json:",omitempty"`
	Optional interface{} `json:",omitempty"`
}

func (p *CallMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	if p.Object != nil {
		amf.writeMap(p.Object.(Map))
	}
	if p.Optional != nil {
		amf.writeMap(p.Optional.(Map))
	}
	p.Payload = amf.Bytes()
}

func (p *CallMessage) Encode3() {
	p.Encode0()
	buf := new(bytes.Buffer)
	buf.WriteByte(0)
	buf.Write(p.Payload)
	p.Payload = buf.Bytes()
}

func (p *CallMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *CallMessage) Body() Payload {
	return p.Payload
}

func (m *CallMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("CallMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyCallMessage struct {
	CommandMessage
	Object   interface{}
	Response interface{}
}

func (p *ReplyCallMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	if p.Object != nil {
		amf.writeMap(p.Object.(Map))
	}
	if p.Response != nil {
		amf.writeMap(p.Response.(Map))
	}
	p.Payload = amf.Bytes()
}

//func (p *ReplyCallMessage) Encode3() {
//	p.Encode0()
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *ReplyCallMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyCallMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyCallMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyCallMessage(%d) %v", len(m.Payload), string(b))
}

type CreateStreamMessage struct {
	CommandMessage
	Object interface{}
}

func (p *CreateStreamMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	amf.writeNull()
	if p.Object != nil {
		amf.writeMap(p.Object.(Map))
	}
	p.Payload = amf.Bytes()
}

//func (p *CreateStreamMessage) Encode3() {
//	p.Encode0()
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *CreateStreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *CreateStreamMessage) Body() Payload {
	return p.Payload
}

func (m *CreateStreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("CreateStreamMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyCreateStreamMessage struct {
	CommandMessage
	Object   interface{} `json:",omitempty"`
	StreamId uint32
}

func (p *ReplyCreateStreamMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	amf.writeNull()
	amf.writeNumber(float64(p.StreamId))
	p.Payload = amf.Bytes()
}

//func (p *ReplyCreateStreamMessage) Encode3() {
//	p.Encode0()
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *ReplyCreateStreamMessage) Decode0(head *RtmpHeader, body Payload) {
	amf := newDecoder(body)
	if obj, err := amf.readData(); err == nil {
		p.Command = obj.(string)
	}
	if obj, err := amf.readData(); err == nil {
		p.TransactionId = uint64(obj.(float64))
	}
	amf.readData()
	if obj, err := amf.readData(); err == nil {
		p.StreamId = uint32(obj.(float64))
	}
}

func (p *ReplyCreateStreamMessage) Decode3(head *RtmpHeader, body Payload) {
	p.Decode0(head, body[1:])
}

func (p *ReplyCreateStreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyCreateStreamMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyCreateStreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyCreateStreamMessage(%d) %v", len(m.Payload), string(b))
}

type PlayMessage struct {
	CommandMessage
	Object     interface{} `json:",omitempty"`
	StreamName string
	Start      uint64
	Duration   uint64
	Rest       bool
}

func (p *PlayMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	amf.writeNull()
	amf.writeString(p.StreamName)
	if p.Start > 0 {
		amf.writeNumber(float64(p.Start))
	}
	if p.Duration > 0 {
		amf.writeNumber(float64(p.Duration))
	}
	amf.writeBool(p.Rest)
	p.Payload = amf.Bytes()
}

//func (p *PlayMessage) Encode3() {
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	p.Encode0()
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *PlayMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *PlayMessage) Body() Payload {
	return p.Payload
}

func (m *PlayMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("PlayMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyPlayMessage struct {
	CommandMessage
	Object      interface{} `json:",omitempty"`
	Description string
}

func (p *ReplyPlayMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	amf.writeNull()
	if p.Object != nil {
		amf.writeMap(p.Object.(Map))
	}
	amf.writeString(p.Description)
	p.Payload = amf.Bytes()
}

//func (p *ReplyPlayMessage) Encode3() {
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	p.Encode0()
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *ReplyPlayMessage) Decode0(head *RtmpHeader, body Payload) {
	//fmt.Printf("%v\n", string(body))
	amf := newDecoder(body)
	if obj, err := amf.readData(); err == nil {
		p.Command = obj.(string)
	}
	if obj, err := amf.readData(); err == nil {
		p.TransactionId = uint64(obj.(float64))
	}
	obj, err := amf.readData()
	//fmt.Printf("%v\n", obj)
	if err == nil && obj != nil {
		p.Object = obj
	} else if obj, err := amf.readData(); err == nil {
		//fmt.Printf("%v\n", obj)
		p.Object = obj
	}
}

func (p *ReplyPlayMessage) Decode3(head *RtmpHeader, body Payload) {
	p.Decode0(head, body[1:])
}

func (p *ReplyPlayMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyPlayMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyPlayMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyPlayMessage(%d) %v", len(m.Payload), string(b))
}

type Play2Message struct {
	CommandMessage
	StartTime     uint64
	OldStreamName string
	StreamName    string
	Duration      uint64
	Transition    string
}

func (p *Play2Message) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNumber(enc, p.StartTime)
	//writeString(enc, p.OldStreamName)
	//writeString(enc, p.StreamName)
	//writeNumber(enc, p.Duration)
	//writeString(enc, p.Transition)
	//p.Payload = buf.Bytes()
}

//func (p *Play2Message) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNumber(enc, p.StartTime)
//	//writeString(enc, p.OldStreamName)
//	//writeString(enc, p.StreamName)
//	//writeNumber(enc, p.Duration)
//	//writeString(enc, p.Transition)
//	//p.Payload = buf.Bytes()
//}

func (p *Play2Message) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *Play2Message) Body() Payload {
	return p.Payload
}

func (m *Play2Message) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("Play2Message(%d) %v", len(m.Payload), string(b))
}

type DeleteStreamMessage struct {
	CommandMessage
	Object   interface{}
	StreamId uint32
}

func (p *DeleteStreamMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeNumber(enc, uint64(p.StreamId))
	//p.Payload = buf.Bytes()
}

//func (p *DeleteStreamMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeNumber(enc, uint64(p.StreamId))
//	//p.Payload = buf.Bytes()
//}

func (p *DeleteStreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *DeleteStreamMessage) Body() Payload {
	return p.Payload
}

func (m *DeleteStreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("DeleteStreamMessage(%d) %v", len(m.Payload), string(b))
}

type ReleaseStreamMessage struct {
	CommandMessage
	Object   interface{}
	StreamId uint32
}

func (p *ReleaseStreamMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeNumber(enc, uint64(p.StreamId))
	//p.Payload = buf.Bytes()
}

//func (p *ReleaseStreamMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeNumber(enc, uint64(p.StreamId))
//	//p.Payload = buf.Bytes()
//}

func (p *ReleaseStreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReleaseStreamMessage) Body() Payload {
	return p.Payload
}

func (m *ReleaseStreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReleaseStreamMessage(%d) %v", len(m.Payload), string(b))
}

type CloseStreamMessage struct {
	CommandMessage
	Object   interface{}
	StreamId uint32
}

func (p *CloseStreamMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeNumber(enc, uint64(p.StreamId))
	//p.Payload = buf.Bytes()
}

//func (p *CloseStreamMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeNumber(enc, uint64(p.StreamId))
//	//p.Payload = buf.Bytes()
//}

func (p *CloseStreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *CloseStreamMessage) Body() Payload {
	return p.Payload
}

func (m *CloseStreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("CloseStreamMessage(%d) %v", len(m.Payload), string(b))
}

type ReceiveAudioMessage struct {
	CommandMessage
	Object   interface{}
	BoolFlag bool
}

func (p *ReceiveAudioMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeBool(enc, p.BoolFlag)
	//p.Payload = buf.Bytes()
}

//func (p *ReceiveAudioMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeBool(enc, p.BoolFlag)
//	//p.Payload = buf.Bytes()
//}

func (p *ReceiveAudioMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReceiveAudioMessage) Body() Payload {
	return p.Payload
}

func (m *ReceiveAudioMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReceiveAudioMessage(%d) %v", len(m.Payload), string(b))
}

type ReceiveVideoMessage struct {
	CommandMessage
	Object   interface{} `json:",omitempty"`
	BoolFlag bool
}

func (p *ReceiveVideoMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeBool(enc, p.BoolFlag)
	//p.Payload = buf.Bytes()
}

//func (p *ReceiveVideoMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeBool(enc, p.BoolFlag)
//	//p.Payload = buf.Bytes()
//}

func (p *ReceiveVideoMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReceiveVideoMessage) Body() Payload {
	return p.Payload
}

func (m *ReceiveVideoMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReceiveVideoMessage(%d) %v", len(m.Payload), string(b))
}

type PublishMessage struct {
	CommandMessage
	Object      interface{} `json:",omitempty"`
	PublishName string
	PublishType string
}

func (p *PublishMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeString(enc, p.PublishName)
	//writeString(enc, p.PublishType)
	//p.Payload = buf.Bytes()
}

//func (p *PublishMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeString(enc, p.PublishName)
//	//writeString(enc, p.PublishType)
//	//p.Payload = buf.Bytes()
//}

func (p *PublishMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *PublishMessage) Body() Payload {
	return p.Payload
}

func (m *PublishMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("PublishMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyPublishMessage struct {
	CommandMessage
	Properties interface{} `json:",omitempty"`
	Infomation interface{} `json:",omitempty"`
}

func (p *ReplyPublishMessage) Encode0() {
	amf := newEncoder()
	amf.writeString(p.Command)
	amf.writeNumber(float64(p.TransactionId))
	amf.writeNull()
	if p.Properties != nil {
		amf.writeMap(p.Properties.(Map))
	}
	if p.Infomation != nil {
		amf.writeMap(p.Infomation.(Map))
	}
	p.Payload = amf.Bytes()
}

//func (p *ReplyPublishMessage) Encode3() {
//	p.Encode0()
//	buf := new(bytes.Buffer)
//	buf.WriteByte(0)
//	buf.Write(p.Payload)
//	p.Payload = buf.Bytes()
//}

func (p *ReplyPublishMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyPublishMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyPublishMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyPublishMessage(%d) %v", len(m.Payload), string(b))
}

type SeekMessage struct {
	CommandMessage
	Object       interface{} `json:",omitempty"`
	Milliseconds uint64
}

func (p *SeekMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeNumber(enc, p.Milliseconds)
	//p.Payload = buf.Bytes()
}

//func (p *SeekMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeNumber(enc, p.Milliseconds)
//	//p.Payload = buf.Bytes()
//}

func (p *SeekMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *SeekMessage) Body() Payload {
	return p.Payload
}

func (m *SeekMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("SeekMessage(%d) %v", len(m.Payload), string(b))
}

type ReplySeekMessage struct {
	CommandMessage
	Description string
}

func (p *ReplySeekMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeString(enc, p.Description)
	//p.Payload = buf.Bytes()
}

//func (p *ReplySeekMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeString(enc, p.Description)
//	//p.Payload = buf.Bytes()
//}

func (p *ReplySeekMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplySeekMessage) Body() Payload {
	return p.Payload
}

func (m *ReplySeekMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplySeekMessage(%d) %v", len(m.Payload), string(b))
}

type PauseMessage struct {
	CommandMessage
	Object       interface{}
	Pause        bool
	Milliseconds uint64
}

func (p *PauseMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeNull(enc)
	//writeBool(enc, p.Pause)
	//writeNumber(enc, p.Milliseconds)
	//p.Payload = buf.Bytes()
}

//func (p *PauseMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeNull(enc)
//	//writeBool(enc, p.Pause)
//	//writeNumber(enc, p.Milliseconds)
//	//p.Payload = buf.Bytes()
//}

func (p *PauseMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *PauseMessage) Body() Payload {
	return p.Payload
}

func (m *PauseMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("PauseMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyPauseMessage struct {
	CommandMessage
	Description string
}

func (p *ReplyPauseMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeString(enc, p.Description)
	//p.Payload = buf.Bytes()
}

//func (p *ReplyPauseMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeString(enc, p.Description)
//	//p.Payload = buf.Bytes()
//}

func (p *ReplyPauseMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyPauseMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyPauseMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyPauseMessage(%d) %v", len(m.Payload), string(b))
}

type ReplyMessage struct {
	CommandMessage
	Properties  interface{} `json:",omitempty"`
	Infomation  interface{} `json:",omitempty"`
	Description string
}

func (p *ReplyMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//writeString(enc, p.Command)
	//writeNumber(enc, p.TransactionId)
	//writeString(enc, p.Description)
	//p.Payload = buf.Bytes()
}

//func (p *ReplyMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//writeString(enc, p.Command)
//	//writeNumber(enc, p.TransactionId)
//	//writeString(enc, p.Description)
//	//p.Payload = buf.Bytes()
//}

func (p *ReplyMessage) Decode0(head *RtmpHeader, body Payload) {
	//fmt.Printf("%v\n", string(body))
	amf := newDecoder(body)
	if obj, err := amf.readData(); err == nil {
		p.Command = obj.(string)
	}
	if obj, err := amf.readData(); err == nil {
		p.TransactionId = uint64(obj.(float64))
	}
}

func (p *ReplyMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *ReplyMessage) Body() Payload {
	return p.Payload
}

func (m *ReplyMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("ReplyMessage(%d) %v", len(m.Payload), string(b))
}

type MetadataMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
	Proterties map[string]interface{} `json:",omitempty"`
}

func (p *MetadataMessage) Encode0() {
	//enc, buf := newAMF0Encoder()
	//if p.Proterties != nil {
	//	writeObject(enc, p.Proterties)
	//}
	//p.Payload = buf.Bytes()
}

//func (p *MetadataMessage) Encode3() {
//	//enc, buf := newAMF0Encoder()
//	//buf.WriteByte(0)
//	//if p.Proterties != nil {
//	//	writeObject(enc, p.Proterties)
//	//}
//	//p.Payload = buf.Bytes()
//}

func (p *MetadataMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *MetadataMessage) Body() Payload {
	return p.Payload
}

func (m *MetadataMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("MetadataMessage(%d) %v", len(m.Payload), string(b))
}

type SharedObjectMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
	//Proterties map[string]interface{} `json:",omitempty"`
}

func (p *SharedObjectMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *SharedObjectMessage) Body() Payload {
	return p.Payload
}

func (m *SharedObjectMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("SharedObjectMessage(%d) %v", len(m.Payload), string(b))
}

type AudioMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *AudioMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *AudioMessage) Body() Payload {
	return p.Payload
}

func (m *AudioMessage) String() string {
	return fmt.Sprintf("AudioMessage(%d) %+v", len(m.Payload), m.RtmpHeader)
}

type VideoMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *VideoMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *VideoMessage) Body() Payload {
	return p.Payload
}

func (m *VideoMessage) String() string {

	return fmt.Sprintf("VideoMessage(%d) %+v", len(m.Payload), m.RtmpHeader)
}

type AggregateMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *AggregateMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *AggregateMessage) Body() Payload {
	return p.Payload
}

func (m *AggregateMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("AggregateMessage(%d) %v", len(m.Payload), string(b))
}

type StreamMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *StreamMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *StreamMessage) Body() Payload {
	return p.Payload
}

func (m *StreamMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("StreamMessage(%d) %v", len(m.Payload), string(b))
}

type UnknowCommandMessage struct {
	CommandMessage
	Payload Payload
}

func (p *UnknowCommandMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *UnknowCommandMessage) Body() Payload {
	return p.Payload
}

func (m *UnknowCommandMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("UnknowCommandMessage(%d) %v", len(m.Payload), string(b))
}

type UnknowRtmpMessage struct {
	RtmpHeader *RtmpHeader
	Payload    Payload
}

func (p *UnknowRtmpMessage) Header() *RtmpHeader {
	return p.RtmpHeader
}
func (p *UnknowRtmpMessage) Body() Payload {
	return p.Payload
}

func (m *UnknowRtmpMessage) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("UnknowRtmpMessage(%d) %v", len(m.Payload), string(b))
}
func decodeRtmpMessage1(chunk *RtmpChunk) RtmpMessage {
	head := newRtmpHeader(chunk.chunkid, chunk.timestamp, int(chunk.length), int(chunk.mtype), chunk.streamid, 0)
	if chunk.exttimestamp {
		head.ExtendTimestamp = chunk.timestamp
		head.Timestamp = 0xffffff
	}
	payload := chunk.body.Bytes()
	switch head.MessageType {
	case RTMP_MSG_CHUNK_SIZE:
		m := new(ChunkSizeMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.ChunkSize = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_ABORT:
		m := new(AbortMessage)
		m.RtmpHeader = head
		m.Payload = payload
		//m.ChunkId = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_ACK:
		m := new(AckMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.SequenceNumber = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_USER:
		eventtype := util.BigEndian.Uint16(payload)
		eventdata := payload[2:]
		switch eventtype {
		case RTMP_USER_STREAM_BEGIN:
			m := new(StreamBeginMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			if len(eventdata) >= 4 {
				//服务端在成功地从客户端接收连接命令之后发送本事件，事件ID为0。事件数据是表示开始起作用的流的ID。
				m.StreamId = util.BigEndian.Uint32(eventdata)
			}
			return m
		case RTMP_USER_STREAM_EOF:
			m := new(StreamEOFMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			//服务端通知客户端流结束，4字节的事件数据表示，回放结束的流的ID
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_STREAM_DRY:
			m := new(StreamDryMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			//服务端通知客户端流枯竭，4字节的事件数据表示枯竭流的ID
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_SET_BUFLEN:
			m := new(SetBufferMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.StreamId = util.BigEndian.Uint32(eventdata)
			m.Millisecond = util.BigEndian.Uint32(eventdata[4:])
			return m
		case RTMP_USER_RECORDED:
			m := new(RecordedMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_PING:
			m := new(PingMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.Timestamp = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_PONG:
			m := new(PongMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		case RTMP_USER_BUFFER_END:
			m := new(BufferEndMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		default:
			m := new(UnknownUserMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		}
	case RTMP_MSG_ACK_SIZE:
		m := new(AckWinSizeMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.AckWinsize = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_BANDWIDTH:
		m := new(SetPeerBandwidthMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.AckWinsize = util.BigEndian.Uint32(payload)
		if len(payload) > 4 {
			m.LimitType = payload[4]
		}
		return m
	case RTMP_MSG_EDGE:
		m := new(EdegMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AUDIO:
		m := new(AudioMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_VIDEO:
		m := new(VideoMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_META:
		m := new(MetadataMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_SHARED:
		m := new(SharedObjectMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_CMD:
		return decodeCommand3(head, payload)
	case RTMP_MSG_AMF_META:
		m := new(MetadataMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF_SHARED:
		m := new(SharedObjectMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF_CMD:
		return decodeCommand0(head, payload)
	case RTMP_MSG_AGGREGATE:
		m := new(AggregateMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_MAX:
		m := new(StreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	default:
		m := new(UnknowRtmpMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m

	}
}
func decodeRtmpMessage(head *RtmpHeader, payload Payload) RtmpMessage {
	switch head.MessageType {
	case RTMP_MSG_CHUNK_SIZE:
		m := new(ChunkSizeMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.ChunkSize = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_ABORT:
		m := new(AbortMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.ChunkId = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_ACK:
		m := new(AckMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.SequenceNumber = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_USER:
		eventtype := util.BigEndian.Uint16(payload)
		eventdata := payload[2:]
		switch eventtype {
		case RTMP_USER_STREAM_BEGIN:
			m := new(StreamBeginMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			if len(eventdata) >= 4 {
				//服务端在成功地从客户端接收连接命令之后发送本事件，事件ID为0。事件数据是表示开始起作用的流的ID。
				m.StreamId = util.BigEndian.Uint32(eventdata)
			}
			return m
		case RTMP_USER_STREAM_EOF:
			m := new(StreamEOFMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			//服务端通知客户端流结束，4字节的事件数据表示，回放结束的流的ID
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_STREAM_DRY:
			m := new(StreamDryMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			//服务端通知客户端流枯竭，4字节的事件数据表示枯竭流的ID
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_SET_BUFLEN:
			m := new(SetBufferMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.StreamId = util.BigEndian.Uint32(eventdata)
			m.Millisecond = util.BigEndian.Uint32(eventdata[4:])
			return m
		case RTMP_USER_RECORDED:
			m := new(RecordedMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.StreamId = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_PING:
			m := new(PingMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			m.Timestamp = util.BigEndian.Uint32(eventdata)
			return m
		case RTMP_USER_PONG:
			m := new(PongMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		case RTMP_USER_BUFFER_END:
			m := new(BufferEndMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		default:
			m := new(UnknownUserMessage)
			m.RtmpHeader = head
			m.Payload = payload
			m.EventType = eventtype
			m.EventData = eventdata
			return m
		}
	case RTMP_MSG_ACK_SIZE:
		m := new(AckWinSizeMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.AckWinsize = util.BigEndian.Uint32(payload)
		return m
	case RTMP_MSG_BANDWIDTH:
		m := new(SetPeerBandwidthMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.AckWinsize = util.BigEndian.Uint32(payload)
		if len(payload) > 4 {
			m.LimitType = payload[4]
		}
		return m
	case RTMP_MSG_EDGE:
		m := new(EdegMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AUDIO:
		m := new(AudioMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_VIDEO:
		m := new(VideoMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_META:
		m := new(MetadataMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_SHARED:
		m := new(SharedObjectMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF3_CMD:
		return decodeCommand3(head, payload)
	case RTMP_MSG_AMF_META:
		m := new(MetadataMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF_SHARED:
		m := new(SharedObjectMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_AMF_CMD:
		return decodeCommand0(head, payload)
	case RTMP_MSG_AGGREGATE:
		m := new(AggregateMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	case RTMP_MSG_MAX:
		m := new(StreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m
	default:
		m := new(UnknowRtmpMessage)
		m.RtmpHeader = head
		m.Payload = payload
		return m

	}
}
func readTransactionId(amf *AMF) uint64 {
	v, _ := amf.readNumber()
	return uint64(v)
}
func readString(amf *AMF) string {
	v, _ := amf.readString()
	return v
}
func readNumber(amf *AMF) uint64 {
	v, _ := amf.readNumber()
	return uint64(v)
}
func readBool(amf *AMF) bool {
	v, _ := amf.readBool()
	return v
}

func readObject(amf *AMF) Map {
	v, _ := amf.readObject()
	return v
}
func decodeCommand0(head *RtmpHeader, payload Payload) RtmpMessage {
	amf := newDecoder(payload)
	cmd := readString(amf)
	switch cmd {
	case "connect":
		m := new(ConnectMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		m.Object = readObject(amf)
		m.Optional = readObject(amf)
		return m
	case "call":
		m := new(CallMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		m.Object = readObject(amf)
		m.Optional = readObject(amf)
		return m
	case "createStream":
		m := new(CreateStreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.Object = readObject(amf)
		return m
	case "play":
		m := new(PlayMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.StreamName = readString(amf)
		m.Start = readNumber(amf)
		m.Duration = readNumber(amf)
		m.Rest = readBool(amf)
		return m
	case "play2":
		m := new(Play2Message)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		m.StartTime = readNumber(amf)
		m.OldStreamName = readString(amf)
		m.StreamName = readString(amf)
		m.Duration = readNumber(amf)
		m.Transition = readString(amf)
		return m
	case "publish", "fcpulish":
		m := new(PublishMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.PublishName = readString(amf)
		m.PublishType = readString(amf)
		return m
	case "pause":
		m := new(PauseMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.Pause = readBool(amf)
		m.Milliseconds = readNumber(amf)
		return m
	case "seek":
		m := new(SeekMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.Milliseconds = readNumber(amf)
		return m
	case "deleteStream":
		m := new(DeleteStreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.StreamId = uint32(readNumber(amf))
		return m
	case "releaseStream":
		m := new(ReleaseStreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.StreamId = uint32(readNumber(amf))
		return m
	case "closeStream":
		m := new(CloseStreamMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.StreamId = uint32(readNumber(amf))
		return m
	case "receiveAudio":
		m := new(ReceiveAudioMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.BoolFlag = readBool(amf)
		return m
	case "receiveVideo":
		m := new(ReceiveVideoMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.TransactionId = readTransactionId(amf)
		amf.readNull()
		m.BoolFlag = readBool(amf)
		return m
	case "_result":
		m := new(ReplyMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.Properties = readObject(amf)
		m.Infomation = readObject(amf)
		return m
	case "onStatus":
		m := new(ReplyMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.Properties = readObject(amf)
		m.Infomation = readObject(amf)
		return m
	case "_error":
		m := new(ReplyMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		m.Properties = readObject(amf)
		m.Infomation = readObject(amf)
		return m
	default:
		m := new(UnknowCommandMessage)
		m.RtmpHeader = head
		m.Payload = payload
		m.Command = cmd
		return m
	}
}

func decodeCommand3(head *RtmpHeader, payload Payload) RtmpMessage {
	return decodeCommand0(head, payload[1:])
}

func encodeChunk12(head *RtmpHeader, payload Payload, size int) (cunk Payload, reset Payload, err error) {
	if size > RTMP_MAX_CHUNK_SIZE || payload == nil || len(payload) == 0 {
		err = ChunkError
		return
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_12 + head.ChunkId))
	b := make([]byte, 3)
	util.BigEndian.PutUint24(b, head.Timestamp)
	buf.Write(b)
	util.BigEndian.PutUint24(b, head.MessageLength)
	buf.Write(b)
	buf.WriteByte(head.MessageType)
	b = make([]byte, 4)
	util.LittleEndian.PutUint32(b, uint32(head.StreamId))
	buf.Write(b)
	if head.Timestamp == 0xffffff {
		b := make([]byte, 4)
		util.LittleEndian.PutUint32(b, head.ExtendTimestamp)
		buf.Write(b)
	}
	if len(payload) > size {
		buf.Write(payload[0:size])
		reset = payload[size:]
	} else {
		buf.Write(payload)
	}
	cunk = buf.Bytes()
	return

}

func encodeChunk8(head *RtmpHeader, payload Payload, size int) (cunk Payload, reset Payload, err error) {
	if size > RTMP_MAX_CHUNK_SIZE || payload == nil || len(payload) == 0 {
		err = ChunkError
		return
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_8 + head.ChunkId))
	b := make([]byte, 3)
	util.BigEndian.PutUint24(b, head.Timestamp)
	buf.Write(b)
	util.BigEndian.PutUint24(b, head.MessageLength)
	buf.Write(b)
	buf.WriteByte(head.MessageType)
	//if head.Timestamp == 0xffffff {
	//	b := make([]byte, 4)
	//	util.LittleEndian.PutUint32(b, head.ExtendTimestamp)
	//	buf.Write(b)
	//}
	if len(payload) > size {
		buf.Write(payload[0:size])
		reset = payload[size:]
	} else {
		buf.Write(payload)
	}
	cunk = buf.Bytes()
	return
}

func encodeChunk4(head *RtmpHeader, payload Payload, size int) (cunk Payload, reset Payload, err error) {
	if size > RTMP_MAX_CHUNK_SIZE || payload == nil || len(payload) == 0 {
		err = ChunkError
		return
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_4 + head.ChunkId))
	b := make([]byte, 3)
	util.BigEndian.PutUint24(b, head.Timestamp)
	buf.Write(b)
	//if head.Timestamp == 0xffffff {
	//	b := make([]byte, 4)
	//	util.LittleEndian.PutUint32(b, head.ExtendTimestamp)
	//	buf.Write(b)
	//}
	if len(payload) > size {
		buf.Write(payload[0:size])
		reset = payload[size:]
	} else {
		buf.Write(payload)
	}
	cunk = buf.Bytes()
	return
}

func encodeChunk1(head *RtmpHeader, payload Payload, size int) (cunk Payload, reset Payload, err error) {
	if size > RTMP_MAX_CHUNK_SIZE || payload == nil || len(payload) == 0 {
		err = ChunkError
		return
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(RTMP_CHUNK_HEAD_1 + head.ChunkId))
	//if head.Timestamp == 0xffffff {
	//	b := make([]byte, 4)
	//	util.LittleEndian.PutUint32(b, head.ExtendTimestamp)
	//	buf.Write(b)
	//}
	if len(payload) > size {
		buf.Write(payload[0:size])
		reset = payload[size:]
	} else {
		buf.Write(payload)
	}
	cunk = buf.Bytes()
	return
}
