package rwbased

import (
	"errors"
	"io"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ stack.LinkEndpoint = (*Endpoint)(nil)

// Endpoint implements the interface of stack.LinkEndpoint from io.ReadWriter.
type Endpoint struct {
	// rw is the io.ReadWriter for reading and writing packets.
	rw io.ReadWriter

	// mtu (maximum transmission unit) is the maximum size of a packet.
	mtu uint32

	// caps holds the endpoint capabilities.
	caps stack.LinkEndpointCapabilities

	dispatcher stack.NetworkDispatcher
}

// New returns stack.LinkEndpoint(.*Endpoint) and error.
func New(rw io.ReadWriter, mtu uint32) (*Endpoint, error) {
	if mtu == 0 {
		return nil, errors.New("MTU size is zero")
	}

	if rw == nil {
		return nil, errors.New("RW interface is nil")
	}

	return &Endpoint{
		rw:  rw,
		mtu: mtu,
	}, nil
}

// Attach launches the goroutine that reads packets from io.ReadWriter and
// dispatches them via the provided dispatcher.
func (e *Endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	go e.dispatchLoop()
	e.dispatcher = dispatcher
}

// IsAttached implements stack.LinkEndpoint.IsAttached.
func (e *Endpoint) IsAttached() bool {
	return e.dispatcher != nil
}

// dispatchLoop dispatches packets to upper layer.
func (e *Endpoint) dispatchLoop() {
	for {
		packet := make([]byte, e.mtu)

		n, err := e.rw.Read(packet)
		if err != nil {
			break
		}

		if !e.IsAttached() {
			continue
		}

		var p tcpip.NetworkProtocolNumber
		switch header.IPVersion(packet) {
		case header.IPv4Version:
			p = header.IPv4ProtocolNumber
		case header.IPv6Version:
			p = header.IPv6ProtocolNumber
		}

		e.dispatcher.DeliverNetworkPacket("", "", p, &stack.PacketBuffer{
			Data: buffer.View(packet[:n]).ToVectorisedView(),
		})
	}
}

func (e *Endpoint) writePacket(pkt *stack.PacketBuffer) *tcpip.Error {
	networkHdr := pkt.NetworkHeader().View()
	transportHdr := pkt.TransportHeader().View()
	payload := pkt.Data.ToView()

	buf := buffer.NewVectorisedView(
		len(networkHdr)+len(transportHdr)+len(payload),
		[]buffer.View{networkHdr, transportHdr, payload},
	)

	if _, err := e.rw.Write(buf.ToView()); err != nil {
		return tcpip.ErrInvalidEndpointState
	}

	return nil
}

// WritePacket writes packet back into io.ReadWriter.
func (e *Endpoint) WritePacket(_ *stack.Route, _ *stack.GSO, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) *tcpip.Error {
	return e.writePacket(pkt)
}

// WritePackets writes packets back into io.ReadWriter.
func (e *Endpoint) WritePackets(_ *stack.Route, _ *stack.GSO, pkts stack.PacketBufferList, _ tcpip.NetworkProtocolNumber) (int, *tcpip.Error) {
	n := 0
	for pkt := pkts.Front(); pkt != nil; pkt = pkt.Next() {
		if err := e.writePacket(pkt); err != nil {
			break
		}
		n++
	}
	return n, nil
}

// WriteRawPacket implements stack.LinkEndpoint.WriteRawPacket.
func (e *Endpoint) WriteRawPacket(vv buffer.VectorisedView) *tcpip.Error {
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Data: vv,
	})
	return e.writePacket(pkt)
}

// MTU implements stack.LinkEndpoint.MTU.
func (e *Endpoint) MTU() uint32 {
	return e.mtu
}

// Capabilities implements stack.LinkEndpoint.Capabilities.
func (e *Endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return e.caps
}

// GSOMaxSize returns the maximum GSO packet size.
func (*Endpoint) GSOMaxSize() uint32 {
	return 1 << 15 /* default */
}

// MaxHeaderLength returns the maximum size of the link layer header. Given it
// doesn't have a header, it just returns 0.
func (*Endpoint) MaxHeaderLength() uint16 {
	return 0
}

// LinkAddress returns the link address of this endpoint.
func (*Endpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

// ARPHardwareType implements stack.LinkEndpoint.ARPHardwareType.
func (*Endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

// AddHeader implements stack.LinkEndpoint.AddHeader.
func (e *Endpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {
}

// Wait implements stack.LinkEndpoint.Wait.
func (e *Endpoint) Wait() {}
