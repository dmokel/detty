package detty

import (
	"io"
	"net"
	"sync/atomic"
	"time"
)

var lanchTime = time.Now()

var connID uint32

type dettyConn struct {
	id            uint32
	compress      CompressType
	padding1      uint8
	padding2      uint8
	readBytes     uint32
	writeBytes    uint32
	readPkgNum    uint32
	writePkgNum   uint32
	active        int64
	rTimeout      time.Duration
	wTimeout      time.Duration
	rLastDeadline time.Time
	wLastDeadline time.Time
	local         string
	peer          string
	ss            Session
}

func (c *dettyConn) ID() uint32 {
	return c.id
}

func (c *dettyConn) LocalAddr() string {
	return c.local
}

func (c *dettyConn) RemoteAddr() string {
	return c.peer
}

func (c *dettyConn) incReadPkgNum() {
	atomic.AddUint32(&c.readPkgNum, 1)
}

func (c *dettyConn) incWritePkgNum() {
	atomic.AddUint32(&c.writePkgNum, 1)
}

func (c *dettyConn) UpdateActive() {
	atomic.StoreInt64(&c.active, int64(time.Since(lanchTime)))
}

func (c *dettyConn) GetActive() time.Time {
	return lanchTime.Add(time.Duration(atomic.LoadInt64(&c.active)))
}

func (c *dettyConn) send(interface{}) (int, error) {
	return 0, nil
}

func (c *dettyConn) close(int) {}

func (c *dettyConn) readTimeout() time.Duration {
	return c.rTimeout
}

func (c *dettyConn) setSession(ss Session) {
	c.ss = ss
}

// Pls do not set read deadline for websocket connection.
// gorilla/websocket/conn.go: NextReader will always fail when got a timeout error.
//
// Pls do not set read deadline when using compression.
func (c *dettyConn) SetReadTimeout(rTimeout time.Duration) {
	if rTimeout < 1 {
		panic("@rTimeout < 1")
	}

	c.rTimeout = rTimeout
	if c.wTimeout == 0 {
		c.wTimeout = rTimeout
	}
}

func (c *dettyConn) writeTimeout() time.Duration {
	return c.wTimeout
}

func (c *dettyConn) SetWriteTimeout(wTimeout time.Duration) {
	if wTimeout < 1 {
		panic("@wTimeout < 1")
	}

	c.wTimeout = wTimeout
	if c.rTimeout == 0 {
		c.rTimeout = wTimeout
	}
}

type dettyTCPConn struct {
	dettyConn
	reader io.Reader
	writer io.Writer
	conn   net.Conn
}

func newDettyTCPConn(conn net.Conn) *dettyTCPConn {
	if conn == nil {
		panic("newDettyTCPConn(conn): @conn is nil")
	}
	var localAddr, peerAddr string
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		peerAddr = conn.RemoteAddr().String()
	}

	return &dettyTCPConn{
		conn:   conn,
		reader: io.Reader(conn),
		writer: io.Writer(conn),
		dettyConn: dettyConn{
			id:       atomic.AddUint32(&connID, 1),
			rTimeout: 1e9,
			wTimeout: 1e9,
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
}
