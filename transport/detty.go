package detty

// IEndPoint is a general identity of the server and client
type IEndPoint interface {
	ID() EndPointID
	EndPointType() EndPointType
	IsClosed() bool
	Close()

	RunEventLoop()
}
