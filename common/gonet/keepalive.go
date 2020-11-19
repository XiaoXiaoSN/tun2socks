package gonet

import (
	"net"
	"time"
)

const (
	tcpKeepAlivePeriod = 30 * time.Second
)

// SetKeepAlive sets tcp keepalive for tcp connection.
func SetKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(tcpKeepAlivePeriod)
	}
}
