package detty

type (
	EndPointID   = int32
	EndPointType int32
)

const (
	// UDP_ENDPOINT EndPointType = 0
	// UDP_CLIENT   EndPointType = 1
	TCP_CLIENT EndPointType = 2
	// WS_CLIENT    EndPointType = 3
	// WSS_CLIENT   EndPointType = 4
	TCP_SERVER EndPointType = 7
	// WS_SERVER    EndPointType = 8
	// WSS_SERVER   EndPointType = 9
)
