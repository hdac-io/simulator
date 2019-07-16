package net

// Address represents network address
type Address interface{}

// Network represents network
type Network interface {
	Accept() Connection
	Connect(destination Address) Connection
}
