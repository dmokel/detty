package detty

import (
	"fmt"
	"io"
	"net"
	"sync"

	gxbytes "github.com/dubbogo/gost/bytes"
	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
)

const (
	maxReadBufLen = 4 * 1024

	defaultSessionName    = "session"
	defaultTCPSessionName = "tcp-session"
)

// ISession ...
type ISession interface {
	Conn() net.Conn
	IsClosed() bool
	EndPoint() IEndPoint

	// codec
	SetReader(IReader)
	SetWriter(IWriter)

	Close()
}

type session struct {
	connection IConnection
	name       string
	endPoint   IEndPoint

	reader IReader
	writer IWriter

	exitChan chan struct{}

	// goroutines sync
	grNum uatomic.Int32

	lock sync.RWMutex
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

func (s *session) SetReader(reader IReader) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reader = reader
}

func (s *session) SetWriter(writer IWriter) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.writer = writer
}

func (s *session) sessionToken() string {
	if s.IsClosed() || s.connection == nil {
		return "session-closed"
	}

	return fmt.Sprintf("{%s:%d:%d:%s<->%s}",
		s.name, s.EndPoint().EndPointType(), s.connection.ID(), s.connection.LocalAddr(), s.connection.RemoteAddr())
}

func (s *session) run() {
	if s.connection == nil || s.writer == nil {
		errStr := fmt.Sprintf("session{name:%s, conn:%#v, writer:%#v}", s.name, s.connection, s.writer)
		fmt.Println(errStr)
		panic(errStr)
	}

	s.grNum.Add(1)
	go s.handlePackage()
}

func (s *session) handlePackage() {
	var err error

	defer func() {
		// TODO recover
		grNum := s.grNum.Add(-1)
		fmt.Printf("%s, [session.handlePackage] gr will exit now, left gr num %d\n", s.sessionToken(), grNum)

		if err != nil {
			fmt.Printf("%s, [session.handlePackage] error:%+v\n", s.sessionToken(), perrors.WithStack(err))
		}
	}()

	if _, ok := s.connection.(*dettyTCPConn); ok {
		if s.reader == nil {
			errStr := fmt.Sprintf("session{name:%s, conn:%#v, reader:%#v}", s.name, s.connection, s.reader)
			fmt.Println(errStr)
			panic(errStr)
		}

		err = s.handleTCPPackage()
	} else {
		panic(fmt.Sprintf("unknown type session{%#v}", s))
	}
}

func (s *session) handleTCPPackage() error {
	var (
		err    error
		exit   bool
		conn   *dettyTCPConn
		buf    []byte
		bufLen int
		pktBuf *gxbytes.Buffer

		pkg    interface{}
		pkgLen int
	)

	pktBuf = gxbytes.NewBuffer(nil)

	conn = s.connection.(*dettyTCPConn)
	for {
		if s.IsClosed() {
			err = nil
			break
		}

		bufLen = 0
		for {
			buf = pktBuf.WriteNextBegin(maxReadBufLen)
			bufLen, err = conn.recv(buf)
			if err != nil {
				if netError, ok := perrors.Cause(err).(net.Error); ok && netError.Timeout() {
					break
				}
				if perrors.Cause(err) == io.EOF {
					fmt.Printf("%s, session.conn read EOF, client send over, session exit\n", s.sessionToken())
					err = nil
					exit = true
					if bufLen != 0 {
						// as https://github.com/apache/dubbo-getty/issues/77#issuecomment-939652203
						// this branch is impossible. Even if it happens, the bufLen will be zero and the error
						// is io.EOF when getty continues to read the socket.
						exit = false
						fmt.Printf("%s, session.conn read EOF, while the bufLen(%d) is non-zero.\n", s.sessionToken(), bufLen)
					}
					break
				}
				fmt.Printf("%s, [session.conn.read] = error:%+v\n", s.sessionToken(), perrors.WithStack(err))
				exit = true
			}
			break
		}
		if 0 != bufLen {
			pktBuf.WriteNextEnd(bufLen)
			for {
				if pktBuf.Len() <= 0 {
					break
				}
				pkg, pkgLen, err = s.reader.Read(s, pktBuf.Bytes())
				// TODO
				if err != nil {
					break
				}
				if pkg == nil {
					break
				}

				// TODO handle pkg with taskPool
				pktBuf.Next(pkgLen)
			}
		}
		if exit {
			break
		}
	}

	return perrors.WithStack(err)
}
