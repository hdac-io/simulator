package net

// Load represents network payload
type Load = interface{}

// Connection represents virtual public network
type Connection interface {
	Read() Load
	Write(l Load)
	GetAddress() Address
}
