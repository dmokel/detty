package detty

// ServerOption ...
type ServerOption func(*ServerOptions)

// ServerOptions ...
type ServerOptions struct {
	addr string
}
