package detty

import (
	"io"
	"net"

	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
)

var (
	connID uatomic.Uint32
)

// IConnection ...
type IConnection interface {
	ID() uint32
	LocalAddr() string
	RemoteAddr() string

	send(interface{}) (int, error)

	close(int)
	setSession(ISession)
}

type dettyConn struct {
	id uint32

	localAddr  string // local address
	remoteAddr string // remote address
	ss         ISession
}

func (c *dettyConn) ID() uint32 {
	return c.id
}

func (c *dettyConn) LocalAddr() string {
	return c.localAddr
}

func (c *dettyConn) RemoteAddr() string {
	return c.remoteAddr
}

func (c *dettyConn) close(int) {}

func (c *dettyConn) setSession(ss ISession) {
	c.ss = ss
}

type dettyTCPConn struct {
	dettyConn
	reader io.Reader
	writer io.Writer
	conn   net.Conn
}

var _ IConnection = &dettyTCPConn{}

func newDettyTCPConn(conn net.Conn) *dettyTCPConn {
	if conn == nil {
		panic("newGettyTCPConn(conn):@conn is nil")
	}
	var localAddr, remoteAddr string
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		remoteAddr = conn.RemoteAddr().String()
	}

	return &dettyTCPConn{
		conn:   conn,
		reader: io.Reader(conn),
		writer: io.Writer(conn),
		dettyConn: dettyConn{
			id:         connID.Add(1),
			localAddr:  localAddr,
			remoteAddr: remoteAddr,
		},
	}
}

func (d *dettyTCPConn) recv(p []byte) (int, error) {
	var (
		err    error
		length int
	)

	length, err = d.reader.Read(p)
	return length, perrors.WithStack(err)
}

func (d *dettyTCPConn) send(pkg interface{}) (int, error) {
	if p, ok := pkg.([]byte); ok {
		lenght, err := d.writer.Write(p)
		if err == nil {
			// TODO record some connection status
		}
		return lenght, perrors.WithStack(err)
	}

	return 0, perrors.Errorf("illegal @pkg{%#v} type", pkg)
}
