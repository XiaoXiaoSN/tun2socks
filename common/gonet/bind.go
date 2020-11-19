package gonet

import (
	"github.com/Dreamacro/clash/component/dialer"
)

// BindToInterface binds to specific interface to dial.
func BindToInterface(name string) {
	dialer.DialHook = dialer.DialerWithInterface(name)
	dialer.ListenPacketHook = dialer.ListenPacketWithInterface(name)
}
