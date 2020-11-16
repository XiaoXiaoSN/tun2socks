// Package proxy provides implementations of proxy protocols.
package proxy

import (
	"context"
	"net"
	"time"

	"github.com/xjasonlyu/tun2socks/common/adapter"
)

const (
	tcpConnectTimeout = 5 * time.Second
)

type Dialer interface {
	DialContext(context.Context, *adapter.Metadata) (net.Conn, error)
	DialUDP(*adapter.Metadata) (net.PacketConn, error)
}

var _defaultDialer Dialer = &Base{}

// SetDialer sets _defaultDialer.
func SetDialer(d Dialer) {
	_defaultDialer = d
}

// Dial uses _defaultDialer to dial TCP.
func Dial(metadata *adapter.Metadata) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpConnectTimeout)
	defer cancel()
	return _defaultDialer.DialContext(ctx, metadata)
}

// DialContext uses _defaultDialer to dial TCP with context.
func DialContext(ctx context.Context, metadata *adapter.Metadata) (net.Conn, error) {
	return _defaultDialer.DialContext(ctx, metadata)
}

// DialUDP uses _defaultDialer to dial UDP.
func DialUDP(metadata *adapter.Metadata) (net.PacketConn, error) {
	return _defaultDialer.DialUDP(metadata)
}
