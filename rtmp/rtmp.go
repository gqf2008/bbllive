package rtmp

const (
	VERSION     = "1.0.2"
	SERVER_NAME = "babylon"
)

const (
	/* RTMP message types */
	RTMP_MSG_CHUNK_SIZE = 1
	RTMP_MSG_ABORT      = 2
	RTMP_MSG_ACK        = 3
	RTMP_MSG_USER       = 4
	RTMP_MSG_ACK_SIZE   = 5
	RTMP_MSG_BANDWIDTH  = 6
	RTMP_MSG_EDGE       = 7

	RTMP_MSG_AUDIO = 8
	RTMP_MSG_VIDEO = 9

	RTMP_MSG_AMF3_META   = 15
	RTMP_MSG_AMF3_SHARED = 16
	RTMP_MSG_AMF3_CMD    = 17

	RTMP_MSG_AMF_META   = 18
	RTMP_MSG_AMF_SHARED = 19
	RTMP_MSG_AMF_CMD    = 20
	RTMP_MSG_AGGREGATE  = 21
	RTMP_MSG_MAX        = 22

	RTMP_CONNECT        = RTMP_MSG_MAX + 1
	RTMP_DISCONNECT     = RTMP_MSG_MAX + 2
	RTMP_HANDSHAKE_DONE = RTMP_MSG_MAX + 3
	RTMP_MAX_EVENT      = RTMP_MSG_MAX + 4

	/* RMTP control message types */
	RTMP_USER_STREAM_BEGIN = 0
	RTMP_USER_STREAM_EOF   = 1
	RTMP_USER_STREAM_DRY   = 2
	RTMP_USER_SET_BUFLEN   = 3
	RTMP_USER_RECORDED     = 4
	RTMP_USER_PING         = 6
	RTMP_USER_PONG         = 7
	RTMP_USER_UNKNOWN      = 8
	RTMP_USER_BUFFER_END   = 31

	/* Chunk header:
	 *   max 3  basic header
	 * + max 11 message header
	 * + max 4  extended header (timestamp) */
	RTMP_MAX_CHUNK_HEADER = 18

	RTMP_DEFAULT_CHUNK_SIZE = 128
	RTMP_MAX_CHUNK_SIZE     = 64 << 10

	RTMP_CHUNK_HEAD_12   = 0 << 6
	RTMP_CHUNK_HEAD_8    = 1 << 6
	RTMP_CHUNK_HEAD_4    = 2 << 6
	RTMP_CHUNK_HEAD_1    = 3 << 6
	RTMP_CHANNEL_CONTROL = 0x02
	RTMP_CHANNEL_COMMAND = 0x03
	RTMP_CHANNEL_AUDIO   = 0x06
	RTMP_CHANNEL_DATA    = 0x05
	RTMP_CHANNEL_VIDEO   = 0x05
)

//var (
//	handlers = make(map[string]func(*RtmpSession, RtmpMessage))
//)

//func HandeFunc(command string, f func(*RtmpSession, RtmpMessage)) {
//	handlers[command] = f
//}

//func findHandle(command string) func(*RtmpSession, RtmpMessage) {
//	if f, exist := handlers[command]; exist {
//		return f
//	}
//	return nil
//}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// type RtmpMessage struct {
// 	Timestamp   uint64
// 	StreamId    uint32
// 	MessageType MessageType
// 	PreferCid   uint32
// 	Payload     []byte
// }

// func Connect(uri string) (*RtmpSocket, error) {

// }

// type RtmpSocket struct {
// 	conn      net.Conn
// 	uri       string
// 	chunkSize int
// }

// func (s *RtmpSocket) handshake() error {
// 	//client
// 	return nil
// }

// func (s *RtmpSocket) handshake1() error {
// 	return nil
// }

// func (s *RtmpSocket) Write(m ...*RtmpMessage) error {
// 	return nil
// }
// func (s *RtmpSocket) Flush() error {
// 	return nil
// }
// func (s *RtmpSocket) Read() (*RtmpMessage, error) {
// 	return nil, nil
// }
// func (s *RtmpSocket) Close() error {
// 	return nil
// }
