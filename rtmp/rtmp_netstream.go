package rtmp

import (
	"errors"
	"fmt"
	"github.com/manucorporat/try"
	"strings"
	"sync"
	"time"
)

const (
	MODE_PRODUCER = 1
	MODE_CONSUMER = 2
	MODE_PROXY    = 3

	Level_Status                      = "status"
	Level_Error                       = "error"
	Level_Warning                     = "warning"
	NetStatus_OnStatus                = "onStatus"
	NetStatus_Result                  = "_result"
	NetStatus_Error                   = "_error"
	NetConnection_Call_BadVersion     = "NetConnection.Call.BadVersion"     //	"error"	以不能识别的格式编码的数据包。
	NetConnection_Call_Failed         = "NetConnection.Call.Failed"         //	"error"	NetConnection.call 方法无法调用服务器端的方法或命令。
	NetConnection_Call_Prohibited     = "NetConnection.Call.Prohibited"     //	"error"	Action Message Format (AMF) 操作因安全原因而被阻止。 或者是 AMF URL 与 SWF 不在同一个域，或者是 AMF 服务器没有信任 SWF 文件的域的策略文件。
	NetConnection_Connect_AppShutdown = "NetConnection.Connect.AppShutdown" //	"error"	正在关闭指定的应用程序。
	NetConnection_Connect_InvalidApp  = "NetConnection.Connect.InvalidApp"  //	"error"	连接时指定的应用程序名无效。
	NetConnection_Connect_Success     = "NetConnection.Connect.Success"     //"status"	连接尝试成功。
	NetConnection_Connect_Closed      = "NetConnection.Connect.Closed"      //"status"	成功关闭连接。
	NetConnection_Connect_Failed      = "NetConnection.Connect.Failed"      //"error"	连接尝试失败。
	NetConnection_Connect_Rejected    = "NetConnection.Connect.Rejected"

	NetStream_Play_Reset          = "NetStream.Play.Reset"
	NetStream_Play_Start          = "NetStream.Play.Start"
	NetStream_Play_StreamNotFound = "NetStream.Play.StreamNotFound"
	NetStream_Play_Stop           = "NetStream.Play.Stop"
	NetStream_Play_Failed         = "NetStream.Play.Failed"

	NetStream_Play_Switch   = "NetStream.Play.Switch"
	NetStream_Play_Complete = "NetStream.Play.Switch"

	NetStream_Data_Start = "NetStream.Data.Start"

	NetStream_Publish_Start     = "NetStream.Publish.Start"     //"status"	已经成功发布。
	NetStream_Publish_BadName   = "NetStream.Publish.BadName"   //"error"	试图发布已经被他人发布的流。
	NetStream_Publish_Idle      = "NetStream.Publish.Idle"      //"status"	流发布者空闲而没有在传输数据。
	NetStream_Unpublish_Success = "NetStream.Unpublish.Success" //"status"	已成功执行取消发布操作。

	NetStream_Buffer_Empty   = "NetStream.Buffer.Empty"
	NetStream_Buffer_Full    = "NetStream.Buffer.Full"
	NetStream_Buffe_Flush    = "NetStream.Buffer.Flush"
	NetStream_Pause_Notify   = "NetStream.Pause.Notify"
	NetStream_Unpause_Notify = "NetStream.Unpause.Notify"

	NetStream_Record_Start    = "NetStream.Record.Start"    //	"status"	录制已开始。
	NetStream_Record_NoAccess = "NetStream.Record.NoAccess" //	"error"	试图录制仍处于播放状态的流或客户端没有访问权限的流。
	NetStream_Record_Stop     = "NetStream.Record.Stop"     //	"status"	录制已停止。
	NetStream_Record_Failed   = "NetStream.Record.Failed"   //	"error"	尝试录制流失败。

	NetStream_Seek_Failed      = "NetStream.Seek.Failed"      //	"error"	搜索失败，如果流处于不可搜索状态，则会发生搜索失败。
	NetStream_Seek_InvalidTime = "NetStream.Seek.InvalidTime" //"error"	对于使用渐进式下载方式下载的视频，用户已尝试跳过到目前为止已下载的视频数据的结尾或在整个文件已下载后跳过视频的结尾进行搜寻或播放。 message.details 属性包含一个时间代码，该代码指出用户可以搜寻的最后一个有效位置。
	NetStream_Seek_Notify      = "NetStream.Seek.Notify"      //	"status"	搜寻操作完成。

)

type Infomation map[string]interface{}
type Args interface{}

type SharedObject interface {
}

type RtmpNetStream struct {
	conn *RtmpNetConnection
	// metaData           *MediaFrame
	// firstVideoKeyFrame *MediaFrame
	// firstAudioKeyFrame *MediaFrame
	// lastVideoKeyFrame  *MediaFrame
	path         string
	bufferTime   time.Duration
	bufferLength uint64
	bufferLoad   uint64
	lock         sync.RWMutex // guards the following
	sh           ServerHandler
	ch           ClientHandler
	mode         int
	vkfsended    bool
	akfsended    bool
	//mrev_duration uint32
	vsend_time uint32
	asend_time uint32
	notify     chan *int
	obj        *StreamObject
	streamName string
	nsid       int
	err        error
	closed     chan bool
}

func newNetStream(conn *RtmpNetConnection, sh ServerHandler, ch ClientHandler) (s *RtmpNetStream) {
	s = new(RtmpNetStream)
	s.nsid = nsid()
	s.conn = conn
	s.sh = sh
	s.ch = ch
	s.notify = make(chan *int, 30)
	s.closed = make(chan bool)
	return
}

func (s *RtmpNetStream) NsID() int {
	return s.nsid
}
func (s *RtmpNetStream) Name() string {
	return s.streamName
}
func (s *RtmpNetStream) String() string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v %v %v vk:%v ak:%v closed:%v", s.nsid, s.streamName, s.conn.conn.RemoteAddr(), s.vkfsended, s.akfsended, s.isClosed())
}

func (s *RtmpNetStream) Close() error {
	if s.isClosed() {
		return nil
	}
	close(s.closed)
	s.conn.Close()
	s.notifyClosed()
	close(s.notify)
	return nil
}
func (s *RtmpNetStream) isClosed() bool {
	select {
	case _, ok := <-s.closed:
		if !ok {
			return true
		}
	default:
	}
	return false
}
func (s *RtmpNetStream) Notify(idx *int) error {
	if s.isClosed() {
		return errors.New("remote connection " + s.conn.remoteAddr + " closed")
	}
	//var err error
	// s.lock.RLock()
	// err = s.err
	// s.lock.RUnlock()
	// if err != nil {
	// 	return err
	// }

	select {
	case s.notify <- idx:
		return nil
	default:
		log.Warn(s.conn.remoteAddr, "buffer full")
	}
	return nil
}

func (s *RtmpNetStream) writeLoop() {
	log.Info(s.conn.conn.RemoteAddr(), "->", s.conn.conn.LocalAddr(), "writeLoop running")
	defer log.Info(s.conn.conn.RemoteAddr(), "->", s.conn.conn.LocalAddr(), "writeLoop stopped")
	var (
		notify = s.notify
		opened bool
		idx    *int
		obj    *StreamObject
		gop    *MediaGop
		err    error
	)
	obj = s.obj
	for {
		select {
		case idx, opened = <-notify:
			if !opened {
				return
			}
			//log.Info("Notify", s.conn.conn.RemoteAddr(), *idx)

			gop = obj.ReadGop(idx)
			if gop != nil {
				// if s.vkfsended {
				// 	err = write(s.conn, gop.freshChunk.Bytes())
				// 	s.vkfsended = true
				// } else {
				// 	err = write(s.conn, gop.chunk.Bytes())
				// }
				// if err != nil {
				// 	log.Error(s.conn.remoteAddr, err)
				// }
				frames := gop.frames[:]
				for _, frame := range frames {
					//log.Info("=====Frame1", frame, *idx)
					if frame.Type == RTMP_MSG_VIDEO {
						s.vsend_time = frame.Timestamp
						err = s.sendVideo(frame)
					} else if frame.Type == RTMP_MSG_AUDIO {
						s.asend_time = frame.Timestamp
						err = s.sendAudio(frame)
					}
					if err != nil {
						break
					}
				}
				if err == nil {
					err = flush(s.conn)
				}
				//log.Info(s.conn.remoteAddr, "V", s.vsend_time, "ms A", s.asend_time, "ms A-V", int64(s.asend_time)-int64(s.vsend_time), "ms")
			}
		}
	}
}

func (s *RtmpNetStream) StreamObject() *StreamObject {
	return s.obj
}

func (s *RtmpNetStream) play(streamName string, args ...Args) error {
	s.streamName = streamName
	conn := s.conn
	s.mode = MODE_PRODUCER
	sendCreateStream(conn)
	for {
		msg, err := readMessage(conn)
		if err != nil {
			return err
		}

		if m, ok := msg.(*UnknowCommandMessage); ok {
			log.Debug(m)
			continue
		}
		reply := new(ReplyCreateStreamMessage)
		reply.Decode0(msg.Header(), msg.Body())
		log.Debug(reply)
		conn.streamid = reply.StreamId
		break
	}
	sendPlay(conn, streamName, 0, 0, false)
	for {
		msg, err := readMessage(conn)
		if err != nil {
			return err
		}
		if m, ok := msg.(*UnknowCommandMessage); ok {
			log.Debug(m)
			continue
		}
		result := new(ReplyPlayMessage)
		result.Decode0(msg.Header(), msg.Body())
		log.Debug(result)
		code := getString(result.Object, "code")
		if code == NetStream_Play_Reset {
			continue
		} else if code == NetStream_Play_Start {
			break
		} else {
			return errors.New(code)
		}
	}
	sendSetBufferMessage(conn)
	if strings.HasSuffix(conn.app, "/") {
		s.path = conn.app + strings.Split(streamName, "?")[0]
	} else {
		s.path = conn.app + "/" + strings.Split(streamName, "?")[0]
	}

	err := s.notifyPlaying()
	if err != nil {
		return err
	}
	go s.readLoop()
	return nil
}

func (s *RtmpNetStream) BufferTime() time.Duration {
	return s.bufferTime
}
func (s *RtmpNetStream) BytesLoaded() uint64 {
	return s.bufferLoad
}
func (s *RtmpNetStream) BufferLength() uint64 {
	return s.bufferLength
}

// func (s *RtmpNetStream) sendFrame(frame *MediaFrame) error {
// 	if frame.Type == RTMP_MSG_VIDEO {
// 		s.vsend_time = frame.Timestamp
// 		return s.sendVideo(frame)
// 	} else if frame.Type == RTMP_MSG_AUDIO {
// 		s.asend_time = frame.Timestamp
// 		return s.sendAudio(frame)
// 	}
// 	return nil
// }
func (s *RtmpNetStream) sendVideo(video *MediaFrame) error {
	if s.vkfsended {
		return sendVideo(s.conn, video)
	}
	if !video.IFrame() {
		log.Warn("not iframe igone", video)
		return nil
	}
	meta := s.obj.metaData
	if meta != nil {
		//log.Info(s, "Send Metadata ", meta)
		sendMetaData(s.conn, meta)
	}
	fkf := s.obj.firstVideoKeyFrame
	if fkf == nil {
		log.Warn("No Video Configurate Record,Ignore Video ", video)
		return nil
	}

	err := sendFullVideo(s.conn, fkf)
	if err != nil {
		return err
	}
	fakf := s.obj.firstAudioKeyFrame
	err = sendFullAudio(s.conn, fakf)
	if err != nil {
		return err
	}
	s.vkfsended = true
	return sendFullVideo(s.conn, video)
}

func (s *RtmpNetStream) sendAudio(audio *MediaFrame) error {
	if !s.vkfsended {
		return nil
	}
	if s.akfsended {
		return sendAudio(s.conn, audio)
	}

	s.akfsended = true
	return sendFullAudio(s.conn, audio)
}

func (s *RtmpNetStream) sendMetaData(data *MediaFrame) error {
	return sendMetaData(s.conn, data)
}

func (s *RtmpNetStream) readLoop() {
	log.Info(s.conn.conn.RemoteAddr(), "->", s.conn.conn.LocalAddr(), "readLoop running")
	defer log.Info(s.conn.conn.RemoteAddr(), "->", s.conn.conn.LocalAddr(), "readLoop stopped")
	conn := s.conn
	var err error
	var msg RtmpMessage
	for {
		if s.isClosed() {
			break
		}
		try.This(func() {
			msg, err = readMessage(conn)
			if err != nil {
				s.notifyError(err)
				return
			}
			//log.Debug("<<<<< ", msg)
			if msg.Header().MessageLength <= 0 {
				return
			}
			if am, ok := msg.(*AudioMessage); ok {
				//log.Info("AUDIO", ">>>>", am.Header().Timestamp, am.Header().MessageLength)
				sp := NewMediaFrame()
				sp.Timestamp = am.RtmpHeader.Timestamp
				if am.RtmpHeader.Timestamp == 0xffffff {
					sp.Timestamp = am.RtmpHeader.ExtendTimestamp
				}
				sp.Type = am.RtmpHeader.MessageType
				sp.StreamId = am.RtmpHeader.StreamId
				sp.Payload.Write(am.Payload)
				one := am.Payload[0]
				sp.AudioFormat = one >> 4
				sp.SamplingRate = (one & 0x0c) >> 2
				sp.SampleLength = (one & 0x02) >> 1
				sp.AudioType = one & 0x01
				err = s.obj.WriteFrame(sp)
				if err != nil {
					log.Warn(err)
				}
			} else if vm, ok := msg.(*VideoMessage); ok {
				//log.Info("VIDEO", ">>>>", vm.Header().Timestamp, vm.Header().MessageLength)
				sp := NewMediaFrame()
				sp.Timestamp = vm.RtmpHeader.Timestamp
				if vm.RtmpHeader.Timestamp == 0xffffff {
					sp.Timestamp = vm.RtmpHeader.ExtendTimestamp
				}
				sp.Type = vm.RtmpHeader.MessageType
				sp.StreamId = vm.RtmpHeader.StreamId
				sp.Payload.Write(vm.Payload)
				one := vm.Payload[0]
				sp.VideoFrameType = one >> 4
				sp.VideoCodecID = one & 0x0f
				err = s.obj.WriteFrame(sp)
				if err != nil {
					log.Warn(err)
				}
			} else if mm, ok := msg.(*MetadataMessage); ok {
				sp := NewMediaFrame()
				sp.Timestamp = mm.RtmpHeader.Timestamp
				if mm.RtmpHeader.Timestamp == 0xffffff {
					sp.Timestamp = mm.RtmpHeader.ExtendTimestamp
				}
				sp.Type = mm.RtmpHeader.MessageType
				sp.StreamId = mm.RtmpHeader.StreamId
				sp.Payload.Write(mm.Payload)
				err = s.obj.WriteFrame(sp)
				if err != nil {
					log.Warn(err)
				}
				//s.videochan <- sp
				log.Debug("IN", " <<<<<<< ", mm)
			} else if csm, ok := msg.(*CreateStreamMessage); ok {
				log.Debug("IN", " <<<<<<< ", csm)
				conn.streamid = conn.nextStreamId(csm.RtmpHeader.ChunkId)
				err := sendCreateStreamResult(conn, csm.TransactionId)
				if err != nil {
					s.notifyError(err)
					return
				}
			} else if m, ok := msg.(*PublishMessage); ok {
				log.Debug("IN", " <<<<<<< ", m)
				if strings.HasSuffix(conn.app, "/") {
					s.path = conn.app + strings.Split(m.PublishName, "?")[0]

				} else {
					s.path = conn.app + "/" + strings.Split(m.PublishName, "?")[0]
				}
				s.streamName = s.path
				if err := s.notifyPublishing(); err != nil {
					err = sendPublishResult(conn, "error", err.Error())
					if err != nil {
						s.notifyError(err)
					}
					return
				}
				err := sendStreamBegin(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				err = sendPublishStart(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				if s.mode == 0 {
					s.mode = MODE_PRODUCER
				} else {
					s.mode = s.mode | MODE_PRODUCER
				}

			} else if m, ok := msg.(*PlayMessage); ok {
				log.Debug("IN", " <<<<<<< ", m)
				if strings.HasSuffix(s.conn.app, "/") {
					s.path = s.conn.app + strings.Split(m.StreamName, "?")[0]
				} else {
					s.path = s.conn.app + "/" + strings.Split(m.StreamName, "?")[0]
				}
				s.streamName = s.path
				conn.writeChunkSize = 512 //RTMP_MAX_CHUNK_SIZE
				err = s.notifyPlaying()
				if err != nil {
					err = sendPlayResult(conn, "error", err.Error())
					if err != nil {
						s.notifyError(err)
					}
					return
				}
				err := sendChunkSize(conn, uint32(conn.writeChunkSize))
				if err != nil {
					s.notifyError(err)
					return
				}
				err = sendStreamRecorded(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				err = sendStreamBegin(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				err = sendPlayReset(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				err = sendPlayStart(conn)
				if err != nil {
					s.notifyError(err)
					return
				}
				if s.mode == 0 {
					s.mode = MODE_CONSUMER
				} else {
					s.mode = s.mode | MODE_CONSUMER
				}
			} else if m, ok := msg.(*CloseStreamMessage); ok {
				log.Debug("IN", " <<<<<<< ", m)
				s.Close()
			} else {
				log.Debug("IN Why?", " <<<<<<< ", msg)
			}
		}).Catch(func(e try.E) {
			log.Error(e)
			s.Close()
		})
	}
}

func (s *RtmpNetStream) notifyPublishing() error {
	if s.sh != nil {
		return s.sh.OnPublishing(s)
	}
	if s.ch != nil {
		return s.ch.OnPublishStart(s)
	}
	return errors.New("Handler Not Found")
}

func (s *RtmpNetStream) notifyPlaying() error {
	if s.sh != nil {
		return s.sh.OnPlaying(s)
	}
	if s.ch != nil {
		return s.ch.OnPlayStart(s)
	}
	return errors.New("Handler Not Found")
}

func (s *RtmpNetStream) notifyError(err error) {
	if s.sh != nil {
		s.sh.OnError(s, err)
	}
	if s.ch != nil {
		s.ch.OnError(s, err)
	}
}

func (s *RtmpNetStream) notifyClosed() {
	if s.sh != nil {
		s.sh.OnClosed(s)
	}
	if s.ch != nil {
		s.ch.OnClosed(s)
	}
}
