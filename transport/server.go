package detty

import (
	"fmt"
	"net"
	"sync"
	"time"

	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
)

var (
	serverID uatomic.Int32
)

// IServer is a general server interface
type IServer interface {
	IEndPoint
}

// IStreamServer is the tcp/websocket/wss server's interface
type IStreamServer interface {
	IServer
	Listener() net.Listener
}

type server struct {
	ServerOptions

	endPointID   EndPointID
	endPointType EndPointType

	streamListener net.Listener

	once     sync.Once
	exitChan chan struct{}
	wg       sync.WaitGroup
}

var _ IServer = &server{}

func (s *server) init(opts ...ServerOption) {
	for _, opt := range opts {
		opt(&(s.ServerOptions))
	}
}

func newServer(endPointType EndPointType, opts ...ServerOption) *server {
	s := &server{
		endPointID:   serverID.Add(1),
		endPointType: endPointType,

		exitChan: make(chan struct{}),
	}

	s.init(opts...)

	return s
}

// NewTCPServer ...
func NewTCPServer(opts ...ServerOption) IServer {
	return newServer(TCP_SERVER, opts...)
}

func (s *server) ID() EndPointID {
	return s.endPointID
}

func (s *server) EndPointType() EndPointType {
	return s.endPointType
}

func (s *server) IsClosed() bool {
	select {
	case <-s.exitChan:
		return true
	default:
		return false
	}
}

func (s *server) stop() {
	select {
	case <-s.exitChan:
		return
	default:
		s.once.Do(func() {
			close(s.exitChan)
			if s.streamListener != nil {
				s.streamListener.Close()
				s.streamListener = nil
			}
		})
	}
}

func (s *server) Close() {
	s.stop()
	s.wg.Wait()
}

func (s *server) listen() error {
	switch s.endPointType {
	case TCP_SERVER:
		return perrors.WithStack(s.listenTCP())
	}
	return nil
}

func (s *server) listenTCP() error {
	streamListener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return perrors.Wrapf(err, "net.Listen(tcp, addr:%s)", s.addr)
	}

	s.streamListener = streamListener
	s.addr = s.streamListener.Addr().String()
	return nil
}

func (s *server) RunEventLoop() {
	if err := s.listen(); err != nil {
		panic(fmt.Errorf("server.listen() = error:%+v", perrors.WithStack(err)))
	}

	switch s.endPointType {
	case TCP_SERVER:
		s.runTCPEventLoop()
	}
}

func (s *server) runTCPEventLoop() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		var (
			err error
			// isession
			delay time.Duration
		)
		for {
			if s.IsClosed() {
				fmt.Printf("server{%s} stop accepting client connect request.\n", s.addr)
				return
			}
			if delay != 0 {
				<-time.After(delay)
			}
			// TODO accept return isession and error
			err = s.accept()
			if err != nil {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if max := 1 * time.Second; delay > max {
					delay = max
				}
				continue
			}
			delay = 0
			// TODO
			// session.run()
		}
	}()
}

// TODO return isession and error
func (s *server) accept() error {
	return nil
}
