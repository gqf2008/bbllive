package rtmp

import (
	"bblgame/libs"
	"flag"
	"github.com/sdming/gosnow"
	cmap "github.com/streamrail/concurrent-map"
	"net"
	"runtime"
	"sync"
	"time"
)

var (
	objects  = cmap.New()
	log      *libs.FileLogger
	shandler ServerHandler = new(DefaultServerHandler)
	logfile  string
	level    int
	snow     *gosnow.SnowFlake
	srvid    int
)

func init() {
	flag.StringVar(&logfile, "log", "stdout", "-log rtmp.log")
	flag.IntVar(&level, "level", 1, "-level 1")
	flag.IntVar(&srvid, "srvid", 1, "-srvid 1")
}
func ListenAndServe(addr string) error {
	logger, err := libs.NewFileLogger("", logfile, level)
	if err != nil {
		return err
	}
	log = logger
	snow, err = gosnow.NewSnowFlake(uint32(srvid))
	if err != nil {
		return err
	}
	srv := &Server{
		Addr:         addr,
		ReadTimeout:  time.Duration(time.Second * 30),
		WriteTimeout: time.Duration(time.Second * 30),
		Lock:         new(sync.Mutex)}
	return srv.ListenAndServe()
}

type Server struct {
	Addr         string        //监听地址
	ReadTimeout  time.Duration //读超时
	WriteTimeout time.Duration //写超时
	Lock         *sync.Mutex
}

func nsid() int {
	id, _ := snow.Next()
	return int(id)
}

var gstreamid = uint32(64)

func gen_next_stream_id(chunkid uint32) uint32 {
	gstreamid += 1
	return gstreamid
}

func (p *Server) ListenAndServe() error {
	addr := p.Addr
	if addr == "" {
		addr = ":1935"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go p.loop(l)
	}
	return nil
}

func (srv *Server) loop(l net.Listener) error {
	defer l.Close()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		grw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Errorf("rtmp: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		go serve(srv, grw)
	}
}

func serve(srv *Server, con net.Conn) {
	log.Info("Accept", con.RemoteAddr(), "->", con.LocalAddr())
	con.(*net.TCPConn).SetNoDelay(true)
	conn := newconn(con, srv)
	if !handshake1(conn.buf) {
		conn.Close()
		return
	}
	log.Info("handshake", con.RemoteAddr(), "->", con.LocalAddr(), "ok")
	log.Debug("readMessage")
	msg, err := readMessage(conn)
	if err != nil {
		log.Error("NetConnecton read error", err)
		conn.Close()
		return
	}

	cmd, ok := msg.(*ConnectMessage)
	if !ok || cmd.Command != "connect" {
		log.Error("NetConnecton Received Invalid ConnectMessage ", msg)
		conn.Close()
		return
	}
	conn.app = getString(cmd.Object, "app")

	conn.objectEncoding = int(getNumber(cmd.Object, "objectEncoding"))
	log.Debug(cmd)
	log.Info(con.RemoteAddr(), "->", con.LocalAddr(), cmd, conn.app, conn.objectEncoding)
	err = sendAckWinsize(conn, 512<<10)
	if err != nil {
		log.Error("NetConnecton sendAckWinsize error", err)
		conn.Close()
		return
	}
	err = sendPeerBandwidth(conn, 512<<10)
	if err != nil {
		log.Error("NetConnecton sendPeerBandwidth error", err)
		conn.Close()
		return
	}
	err = sendStreamBegin(conn)
	if err != nil {
		log.Error("NetConnecton sendStreamBegin error", err)
		conn.Close()
		return
	}
	err = sendConnectSuccess(conn)
	if err != nil {
		log.Error("NetConnecton sendConnectSuccess error", err)
		conn.Close()
		return
	}
	conn.connected = true
	newNetStream(conn, shandler, nil).readLoop()
}

func getNumber(obj interface{}, key string) float64 {
	if v, exist := obj.(Map)[key]; exist {
		return v.(float64)
	}
	return 0.0
}

func findObject(name string) (*StreamObject, bool) {
	if v, found := objects.Get(name); found {
		return v.(*StreamObject), true
	}
	return nil, false
}

func addObject(obj *StreamObject) {
	objects.Set(obj.name, obj)
}

func removeObject(name string) {
	objects.Remove(name)
}
