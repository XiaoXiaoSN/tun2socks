package gonet

import (
	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/resolver"
)

type Resolver = resolver.Resolver

var (
	ResolveIP   = resolver.ResolveIP
	ResolveIPv4 = resolver.ResolveIPv4
	ResolveIPv6 = resolver.ResolveIPv6
)

type Enhancer = resolver.Enhancer

var (
	FakeIPEnabled  = resolver.FakeIPEnabled
	FindHostByIP   = resolver.FindHostByIP
	IsExistFakeIP  = resolver.IsExistFakeIP
	IsFakeIP       = resolver.IsFakeIP
	MappingEnabled = resolver.MappingEnabled
)

var (
	Dial         = dialer.Dial
	Dialer       = dialer.Dialer
	DialContext  = dialer.DialContext
	ListenPacket = dialer.ListenPacket
)
