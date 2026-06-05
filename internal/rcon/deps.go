package rcon

import (
	"net"
	"time"
)

//go:generate mockery --name=Dialer --output=./mocks

// Dialer abstracts TCP connection establishment.
type Dialer interface {
	Dial(addr string, timeout time.Duration) (net.Conn, error)
}

// NetDialer is the production Dialer backed by net.DialTimeout.
type NetDialer struct{}

func (NetDialer) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, timeout)
}
