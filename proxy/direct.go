package proxy

import (
	"context"
	"net"

	"github.com/xjasonlyu/tun2socks/common/adapter"
	"github.com/xjasonlyu/tun2socks/common/gonet"
)

type Direct struct {
	*Base
}

func NewDirect() *Direct {
	return &Direct{}
}

func (d *Direct) DialContext(ctx context.Context, metadata *adapter.Metadata) (net.Conn, error) {
	c, err := gonet.DialContext(ctx, "tcp", metadata.DestinationAddress())
	if err != nil {
		return nil, err
	}
	gonet.SetKeepAlive(c)
	return c, nil
}

func (d *Direct) DialUDP(_ *adapter.Metadata) (net.PacketConn, error) {
	pc, err := gonet.ListenPacket("udp", "")
	if err != nil {
		return nil, err
	}
	return &directPacketConn{PacketConn: pc}, nil
}

type directPacketConn struct {
	net.PacketConn
}

func (pc *directPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if m, ok := addr.(*adapter.Metadata); ok && m.DstIP != nil {
		return pc.PacketConn.WriteTo(b, m.UDPAddr())
	}

	udpAddr, err := gonet.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return 0, err
	}
	return pc.PacketConn.WriteTo(b, udpAddr)
}
