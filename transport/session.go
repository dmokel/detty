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

	outputFormat = "session %s, Read Bytes: %d, Write Bytes: %d, Read Pkgs: %d, Write Pkgs: %d\n"
)

// ISession ...
type ISession interface {
	Conn() net.Conn
	Stat() string
	IsClosed() bool
	EndPoint() IEndPoint

	SetMaxMsgLen(int)
	SetName(string)

	SetEventListener(IEventListener)
	// codec
	SetPkgHandler(IReadWriter)
	SetReader(IReader)
	SetWriter(IWriter)

	// WritePkg: the IWriter will invoke this function.
	// totalBytesLength: @pkg stream bytes length after encoding @pkg.
	// sendBytesLength: stream bytes length sent out successfully.
	// err: maybe it has illegal data, encoding error, or write out system error.
	WritePkg(pkg interface{}) (totalBytesLength int, sendBytesLength int, err error)

	Close()
}

type session struct {
	connection IConnection
	name       string
	endPoint   IEndPoint

	listener IEventListener

	reader    IReader
	writer    IWriter
	maxMsgLen int32

	once     *sync.Once
	exitChan chan struct{}

	// goroutines sync
	grNum uatomic.Int32

	lock       sync.RWMutex
	packetLock sync.RWMutex
}

var _ ISession = &session{}

func newSession(endPoint IEndPoint, conn IConnection) *session {
	ss := &session{
		name:       defaultSessionName,
		endPoint:   endPoint,
		connection: conn,

		maxMsgLen: maxReadBufLen,

		once:     &sync.Once{},
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

func (s *session) incReadPkgNum() {
	if s == nil {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.connection != nil {
		s.connection.incReadPkgNum()
	}
}

func (s *session) incWritePkgNum() {
	if s == nil {
		return
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.connection != nil {
		s.connection.incWritePkgNum()
	}
}

func (s *session) dettyConn() *dettyConn {
	if dc, ok := s.connection.(*dettyTCPConn); ok {
		return &dc.dettyConn
	}
	return nil
}

func (s *session) Stat() string {
	conn := s.dettyConn()
	if conn == nil {
		return ""
	}
	return fmt.Sprintf(
		outputFormat,
		s.sessionToken(),
		conn.readBytes.Load(),
		conn.writeBytes.Load(),
		conn.readPkgNum.Load(),
		conn.writePkgNum.Load(),
	)
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

func (s *session) stop() {
	select {
	case <-s.exitChan:
		return
	default:
		s.once.Do(func() {
			close(s.exitChan)
		})
	}
}

func (s *session) gc() {
	// TODO
}

func (s *session) Close() {
	s.stop()
	fmt.Printf("%s closed now. its current gr num is %d\n", s.sessionToken(), s.grNum.Load())
}

func (s *session) SetMaxMsgLen(length int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.maxMsgLen = int32(length)
}

func (s *session) SetName(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.name = name
}

func (s *session) SetEventListener(listener IEventListener) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.listener = listener
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

func (s *session) SetPkgHandler(handler IReadWriter) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reader = handler
	s.writer = handler
}

func (s *session) sessionToken() string {
	if s.IsClosed() || s.connection == nil {
		return "session-closed"
	}

	return fmt.Sprintf("{%s:%d:%d:%s<->%s}",
		s.name, s.EndPoint().EndPointType(), s.connection.ID(), s.connection.LocalAddr(), s.connection.RemoteAddr())
}

func (s *session) run() {
	if s.connection == nil || s.writer == nil || s.listener == nil {
		errStr := fmt.Sprintf("session{name:%s, conn:%#v, writer:%#v, listener:%#v}", s.name, s.connection, s.writer, s.listener)
		fmt.Println(errStr)
		panic(errStr)
	}

	if err := s.listener.OnOpen(s); err != nil {
		fmt.Printf("[OnOpen] session %s, error: %#v\n", s.Stat(), err)
		s.Close()
		return
	}

	s.grNum.Add(1)
	go s.handlePackage()
}

func (s *session) addTask(pkg interface{}) {
	f := func() {
		s.listener.OnMessage(s, pkg)
		s.incReadPkgNum()
	}

	// TODO handle pkg with task pool

	f()
}

func (s *session) handlePackage() {
	var err error

	defer func() {
		if r := recover(); r != nil {
			// TODO
		}
		grNum := s.grNum.Add(-1)
		fmt.Printf("%s, [session.handlePackage] gr will exit now, left gr num %d\n", s.sessionToken(), grNum)
		s.stop()

		if err != nil {
			fmt.Printf("%s, [session.handlePackage] error:%+v\n", s.sessionToken(), perrors.WithStack(err))
			if s != nil && s.listener != nil {
				s.listener.OnError(s, err)
			}
		}

		s.listener.OnClose(s)
		s.gc()
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
				if err == nil && s.maxMsgLen > 0 && pkgLen > int(s.maxMsgLen) {
					err = perrors.Errorf("pkgLen %d > session max message len %d", pkgLen, s.maxMsgLen)
				}
				if err != nil {
					fmt.Printf("%s, [session.handleTCPPackage] = len{%d}, error:%+v", s.sessionToken(), pkgLen, perrors.WithStack(err))
					exit = true
					break
				}
				if pkg == nil {
					break
				}

				s.addTask(pkg)
				pktBuf.Next(pkgLen)
			}
		}
		if exit {
			break
		}
	}

	return perrors.WithStack(err)
}

func (s *session) WritePkg(pkg interface{}) (totalBytesLength int, sendBytesLength int, err error) {
	if pkg == nil {
		return 0, 0, fmt.Errorf("pkg is nil")
	}
	if s.IsClosed() {
		return 0, 0, ErrSessionClosed
	}

	pkgBytes, err := s.writer.Write(s, pkg)
	if err != nil {
		fmt.Printf("%s, [session.WritePkg] session.writer.Write(@pkg:%#v) = error:%+v\n", s.Stat(), pkg, err)
		return len(pkgBytes), 0, perrors.WithStack(err)
	}
	pkg = pkgBytes

	s.packetLock.RLock()
	defer s.packetLock.RUnlock()

	succCount, err := s.connection.send(pkg)
	if err != nil {
		fmt.Printf("%s, [session.WritePkg] @s.Connection.Write(pkg:%#v) = err:%+v\n", s.Stat(), pkg, err)
		return len(pkgBytes), succCount, perrors.WithStack(err)
	}
	s.incWritePkgNum()
	return len(pkgBytes), succCount, nil
}
