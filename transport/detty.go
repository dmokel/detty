package detty

// NewSessionCallback will be invoked when server accepts a new client connection
type NewSessionCallback func(ISession) error

// IReader ...
type IReader interface {
	// Read Parse tcp/udp/websocket pkg from buffer and if possible return a complete pkg.
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
	Read(ISession, []byte) (interface{}, int, error)
}

// IWriter ...
type IWriter interface {
	Write(ISession, interface{}) ([]byte, error)
}

// IReadWriter interface use for handle application packages
type IReadWriter interface {
	IReader
	IWriter
}

// IEventListener is used to process pkg that received from remote session
type IEventListener interface {
	// OnOpen invoked when session opened
	// If the return error is not nil, @session will be closed.
	OnOpen(ISession) error

	// OnClose invoked when session closed.
	OnClose(ISession)

	// OnError invoked when got error
	OnError(ISession, error)

	// OnMessage invoked when detty received a message, pls attention that do not handle long time
	// logic processing in this function. You'd better set package's maximum length.
	// If the message's length is greater than it, you should return err in
	// IReader{Reade} and detty will close the connection soon.
	//
	// If your logic processing in this function will take a long time, you should start goroutine
	// pool to handle the processing asynchronously. Or you can do the logic processing in other
	// asynchronous way.
	// !!!In short, your OnMessage callback function should return asap.
	OnMessage(ISession, interface{})
}

// IEndPoint is a general identity of the server and client
type IEndPoint interface {
	ID() EndPointID
	EndPointType() EndPointType
	IsClosed() bool
	Close()

	RunEventLoop(newSession NewSessionCallback)
}
