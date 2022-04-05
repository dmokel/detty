package detty

import "strconv"

// EndPointID ...
type EndPointID = int32

// EndPointType ...
type EndPointType int32

const (
	// UDPEndPoint ...
	UDPEndPoint EndPointType = iota
	// UDPClient ...
	UDPClient
	// TCPClient ...
	TCPClient
	// WSClient ...
	WSClient
	// WSSClient ...
	WSSClient
	// TCPServer ...
	TCPServer
	// WSServer ...
	WSServer
	// WSSServer ...
	WSSServer
)

// EndPointTypeName ...
var EndPointTypeName = map[int32]string{
	0: "UDPEndPoint",
	1: "UDPClient",
	2: "TCPClient",
	3: "WSClient",
	4: "WSSClient",
	5: "TCPServer",
	6: "WSServer",
	7: "WSSServer",
}

// EndPointTypeValue ...
var EndPointTypeValue = map[string]int32{
	"UDPEndPoint": 0,
	"UDPClient":   1,
	"TCPClient":   2,
	"WSClient":    3,
	"WSSClient":   4,
	"TCPServer":   5,
	"WSServer":    6,
	"WSSServer":   7,
}

func (x EndPointType) String() string {
	s, ok := EndPointTypeName[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}
