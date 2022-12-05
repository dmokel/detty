package detty

import "net"

const (
	defaultSessionName    = "session"
	defaultTCPSessionName = "tcp-session"
)

// ISession ...
type ISession interface {
	Conn() net.Conn
	IsClosed() bool
	EndPoint() IEndPoint

	Close()
}

type session struct {
	connection IConnection
	name       string
	endPoint   IEndPoint

	exitChan chan struct{}
}

var _ ISession = &session{}

func newSession(endPoint IEndPoint, conn IConnection) *session {
	ss := &session{
		name:       defaultSessionName,
		endPoint:   endPoint,
		connection: conn,

		exitChan: make(chan struct{}),
	}

	ss.connection.setSession(ss)
	return ss
}

// NewTCPSession ...
func newTCPSession(conn net.Conn, endPoint IEndPoint) ISession {
	c := newDettyTCPConn(conn)
	ss := newSession(endPoint, c)
	ss.name = defaultTCPSessionName

	return ss
}

func (s *session) Conn() net.Conn {
	if tc, ok := s.connection.(*dettyTCPConn); ok {
		return tc.conn
	}
	return nil
}

func (s *session) IsClosed() bool {
	select {
	case <-s.exitChan:
		return true
	default:
		return false
	}
}

func (s *session) EndPoint() IEndPoint {
	return s.endPoint
}

func (s *session) Close() {
	// TODO
}
