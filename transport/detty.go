package detty

import (
	"compress/flate"
	"errors"
	"net"
	"time"

	gxsync "github.com/dubbogo/gost/sync"
)

// NewSessionCallback will be invoked when server accepts a new client connection
// or client connects to server successfully.
// If there are too many client connections or u do not want to connect a server again,
// u can return non-nil error. And then detty will close the new session.
type NewSessionCallback func(Session) error

// Reader is used to unmarshal a complete pkg from buffer
type Reader interface {
	// Parse tcp/udp/websocket pkg from buffer and if possible return a complete pkg.
	// When receiving a tcp network streaming segment, there are 4 cases as following:
	// case 1: a error found in the streaming segment;
	// case 2: can not unmarshal a pkg header from the streaming segment;
	// case 3: unmarshal a pkg header but can not unmarshal a pkg from the streaming segment;
	// case 4: just unmarshal a pkg from the streaming segment;
	// case 5: unmarshal more than one pkg from the streaming segment;
	//
	// The return value is (nil, 0, error) as case 1.
	// The return value is (nil, 0, nil) as case 2.
	// The return value is (nil, pkgLen, nil) as case 3.
	// The return value is (pkg, pkgLen, nil) as case 4.
	// The handleTcpPackage may invoke func Read many times as case 5.
	Read(Session, []byte) (interface{}, int, error)
}

// Writer is used to marshal pkg and write to session
type Writer interface {
	// if @Session is udpDettySession, the second parameter is UDPContext.
	Write(Session, interface{}) ([]byte, error)
}

// ReadWrite ...
type ReadWrite interface {
	Reader
	Writer
}

// EventListener is used to process pkg that received from remote session.
type EventListener interface {
	// invoked when session opened, if the return error is not nil, @Session will be closed.
	OnOpen(Session) error
	// invoked when session closed.
	OnClose(Session)
	// invoked when got error
	OnError(Session, error)
	// invoked periodically, its period can be set by (Session)SetCronPeriod
	OnCron(Session)
	// invoked when detty received a package. Pls attention that do not handle long time
	// logic processing in this func. You'd better set the package's maximum length.
	// If the message's length is greater than it, u should return err in Reader{Read}
	// and detty will close this connection soon.
	//
	// If ur logic processing in this func will take a long time, u should start a goroutine
	// pool to handle the processing asynchronously, or u can do the logic processing in other
	// asynchronous way.
	// !!! In short, ur OnMessage callback func should return asap.
	//
	// If this is a udp event listener, the second parameter type is UDPContext.
	OnMessage(Session, interface{})
}

type CompressType int

const (
	CompressNone           CompressType = flate.NoCompression
	CompressZip                         = flate.DefaultCompression
	CompressBestSpeed                   = flate.BestSpeed
	CompressBestCompressin              = flate.BestCompression
	CompressHuffman                     = flate.HuffmanOnly
	CompressSnappy                      = 10
)

// Conn is the abstract net connection on detty
type Conn interface {
	ID() uint32
	SetCompressType(CompressType)
	LocalAddr() string
	RemoteAddr() string
	incReadPkgNum()
	incWritePkgNum()
	UpdateActive()        // update session's active time
	GetActive() time.Time // get session's active time
	readTimeout() time.Duration
	SetReadTimeout(time.Duration) // SetReadTimeout sets deadline for the future read calls.
	writeTimeout() time.Duration
	SetWriteTimeout(time.Duration) // SetWriteTimeout sets deadline for the future read calls.
	send(interface{}) (int, error)
	// don't distinguish between tcp connection and websocket connection. Because
	// gorilla/websocket/conn.go:(Conn)Close also invoke net.Conn.Close
	close(int)
	setSession()
}

var (
	ErrSessionClosed  = errors.New("session Already Closed")
	ErrSessionBlocked = errors.New("session Full Blocked")
	ErrNullPeerAddr   = errors.New("peer address is nil")
)

// Session ...
type Session interface {
	Conn
	Reset()
	Conn() net.Conn
	Stat() string
	IsClosed() bool
	EndPoint() EndPoint // get endpoint type

	SetMaxMsgLen(int)
	SetName(string)
	SetEventListener(EventListener)
	SetPkgHandler(ReadWrite)
	SetReader(Reader)
	SetWriter(Writer)
	SetCronPeriod(int)

	SetWQlen(int)
	SetWaitTime(time.Duration)

	GetAttribute(interface{}) interface{}
	SetAttribute(interface{}, interface{})
	RemoveAttribute(interface{})

	// the writer will invoke this function. Pls attention that if timeout is less than 0,
	// WritePkg will send @Pkg asap. For udp session, the first parameter should be UDPContext.
	WritePkg(pkg interface{}, timeout time.Duration)
	WriteBytes([]byte) error
	WriteBytesArray(...[]byte) error
	Close()
}

// EndPoint ...
type EndPoint interface {
	ID() EndPointID
	EndPointType() EndPointType
	RunEventLoop(newSession NewSessionCallback)
	IsClosed() bool
	Close()
	GetTaskPool() gxsync.GenericTaskPool
}

// Client ...
type Client interface {
	EndPoint
}

// Server ...
type Server interface {
	EndPoint
}

// StreamServer is like tcp/websocket/wss server
type StreamServer interface {
	Server
	Listener() net.Listener
}

// PacketServer is like udp listen endpoint
type PacketServer interface {
	Server
	PacketConn() net.PacketConn
}
