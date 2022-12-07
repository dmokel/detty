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

// IEndPoint is a general identity of the server and client
type IEndPoint interface {
	ID() EndPointID
	EndPointType() EndPointType
	IsClosed() bool
	Close()

	RunEventLoop(newSession NewSessionCallback)
}
