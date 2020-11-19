package gonet

import (
	"net"

	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/component/trie"
)

func init() {
	// use bound dialer to resolve DNS
	net.DefaultResolver.Dial = DialContext
	net.DefaultResolver.PreferGo = true
}

// EnableIPv6 enables/disables ipv6 for resolver.
func EnableIPv6(v bool) {
	resolver.DisableIPv6 = !v
}

// SetHosts sets default hosts.
func SetHosts(h *trie.DomainTrie) {
	resolver.DefaultHosts = h
}

// SetHostMapper sets default host mapper.
func SetHostMapper(m Enhancer) {
	resolver.DefaultHostMapper = m
}

// SetResolver sets default resolver.
func SetResolver(r Resolver) {
	resolver.DefaultResolver = r
}

// ResolveUDPAddr resolves address to *net.UDPAddr.
func ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	ip, err := ResolveIP(host)
	if err != nil {
		return nil, err
	}

	return net.ResolveUDPAddr(network, net.JoinHostPort(ip.String(), port))
}
