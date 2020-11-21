package config

// Ref: https://github.com/Dreamacro/clash/blob/master/config/config.go

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/xjasonlyu/tun2socks/device"
	"github.com/xjasonlyu/tun2socks/device/tun"
	"github.com/xjasonlyu/tun2socks/log"
	"github.com/xjasonlyu/tun2socks/proxy"

	"github.com/Dreamacro/clash/component/fakeip"
	"github.com/Dreamacro/clash/component/trie"
	"github.com/Dreamacro/clash/dns"
	"gopkg.in/yaml.v2"
)

type Config struct {
	General *General
	DNS     *DNS
	Hosts   *trie.DomainTrie
}

type General struct {
	Device    device.Device
	Proxy     proxy.Dialer
	LogLevel  log.Level
	Interface string
	Secret    string
	Stats     string
	IPv6      bool
}

type DNS struct {
	Enable            bool
	IPv6              bool
	NameServer        []dns.NameServer
	Listen            string
	EnhancedMode      dns.EnhancedMode
	DefaultNameserver []dns.NameServer
	FakeIPRange       *fakeip.Pool
	Hosts             *trie.DomainTrie
}

type RawConfig struct {
	Device    string    `yaml:"device"`
	Interface string    `yaml:"interface"`
	LogLevel  log.Level `yaml:"log-level"`
	MTU       int       `yaml:"mtu"`
	Proxy     string    `yaml:"proxy"`
	Secret    string    `yaml:"secret"`
	Stats     string    `yaml:"stats"`
	IPv6      bool      `yaml:"ipv6"`

	DNS   RawDNS            `yaml:"dns"`
	Hosts map[string]string `yaml:"hosts"`
}

type RawDNS struct {
	Enable            bool             `yaml:"enable"`
	IPv6              bool             `yaml:"ipv6"`
	UseHosts          bool             `yaml:"use-hosts"`
	NameServer        []string         `yaml:"nameserver"`
	Listen            string           `yaml:"listen"`
	EnhancedMode      dns.EnhancedMode `yaml:"enhanced-mode"`
	FakeIPRange       string           `yaml:"fake-ip-range"`
	FakeIPFilter      []string         `yaml:"fake-ip-filter"`
	DefaultNameserver []string         `yaml:"default-nameserver"`
}

func Parse(buf []byte) (*Config, error) {
	rawCfg, err := UnmarshalRawConfig(buf)
	if err != nil {
		return nil, err
	}

	return ParseRawConfig(rawCfg)
}

func UnmarshalRawConfig(buf []byte) (*RawConfig, error) {
	// config with some default values.
	rawCfg := &RawConfig{
		IPv6:     false,
		LogLevel: log.InfoLevel,
		Hosts:    map[string]string{},
		DNS: RawDNS{
			Enable:       false,
			IPv6:         true,
			UseHosts:     true,
			EnhancedMode: dns.FAKEIP,
			FakeIPRange:  "198.18.0.0/15",
			DefaultNameserver: []string{
				"223.5.5.5",
				"8.8.8.8",
			},
		},
	}

	if err := yaml.Unmarshal(buf, &rawCfg); err != nil {
		return nil, err
	}

	return rawCfg, nil
}

func ParseRawConfig(rawCfg *RawConfig) (*Config, error) {
	config := &Config{}

	general, err := parseGeneral(rawCfg)
	if err != nil {
		return nil, err
	}
	config.General = general

	hosts, err := parseHosts(rawCfg)
	if err != nil {
		return nil, err
	}
	config.Hosts = hosts

	dnsCfg, err := parseDNS(rawCfg.DNS, hosts)
	if err != nil {
		return nil, err
	}
	config.DNS = dnsCfg

	return config, nil
}

func parseGeneral(cfg *RawConfig) (*General, error) {
	_device, err := parseDevice(cfg.Device, cfg.MTU)
	if err != nil {
		return nil, err
	}

	_proxy, err := parseProxy(cfg.Proxy)
	if err != nil {
		return nil, err
	}

	return &General{
		Device:    _device,
		Proxy:     _proxy,
		Stats:     cfg.Stats,
		Secret:    cfg.Secret,
		LogLevel:  cfg.LogLevel,
		IPv6:      cfg.IPv6,
		Interface: cfg.Interface,
	}, nil
}

func parseDevice(raw string, mtu int) (device.Device, error) {
	const defaultScheme = "tun"
	if !strings.Contains(raw, "://") {
		raw = defaultScheme + "://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(u.Scheme) {
	case "tun":
		name := u.Host
		return tun.Open(tun.WithName(name), tun.WithMTU(uint32(mtu)))
	default:
		// reserved
	}

	return nil, fmt.Errorf("unsupported device type: %s", u.Scheme)
}

func parseProxy(raw string) (proxy.Dialer, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	proto := strings.ToLower(u.Scheme)
	addr := u.Host
	user := u.User.Username()
	pass, _ := u.User.Password()

	switch proto {
	case "direct":
		return proxy.NewDirect(), nil
	case "socks5":
		return proxy.NewSocks5(addr, user, pass)
	case "ss", "shadowsocks":
		method, password := user, pass
		return proxy.NewShadowSocks(addr, method, password)
	}

	return nil, fmt.Errorf("unsupported protocol: %s", proto)
}

func parseHosts(cfg *RawConfig) (*trie.DomainTrie, error) {
	tree := trie.New()

	// add default hosts
	if err := tree.Insert("localhost", net.IP{127, 0, 0, 1}); err != nil {
		log.Errorf("insert localhost to host error: %s", err.Error())
	}

	if len(cfg.Hosts) != 0 {
		for domain, ipStr := range cfg.Hosts {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("%s is not a valid IP", ipStr)
			}
			if err := tree.Insert(domain, ip); err != nil {
				return nil, err
			}
		}
	}

	return tree, nil
}

func hostWithDefaultPort(host string, defPort string) (string, error) {
	if !strings.Contains(host, ":") {
		host += ":"
	}

	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", err
	}

	if port == "" {
		port = defPort
	}

	return net.JoinHostPort(hostname, port), nil
}

func parseNameServer(servers []string) ([]dns.NameServer, error) {
	var nameservers []dns.NameServer

	for idx, server := range servers {
		// parse without scheme .e.g 8.8.8.8:53
		if !strings.Contains(server, "://") {
			server = "udp://" + server
		}
		u, err := url.Parse(server)
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		var addr, dnsNetType string
		switch u.Scheme {
		case "udp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "" // UDP
		case "tcp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "tcp" // TCP
		case "tls":
			addr, err = hostWithDefaultPort(u.Host, "853")
			dnsNetType = "tcp-tls" // DNS over TLS
		case "https":
			clearURL := url.URL{Scheme: "https", Host: u.Host, Path: u.Path}
			addr = clearURL.String()
			dnsNetType = "https" // DNS over HTTPS
		default:
			return nil, fmt.Errorf("DNS NameServer[%d] unsupport scheme: %s", idx, u.Scheme)
		}

		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		nameservers = append(
			nameservers,
			dns.NameServer{
				Net:  dnsNetType,
				Addr: addr,
			},
		)
	}
	return nameservers, nil
}

func parseDNS(cfg RawDNS, hosts *trie.DomainTrie) (*DNS, error) {
	if cfg.Enable && len(cfg.NameServer) == 0 {
		return nil, fmt.Errorf("if DNS configuration is turned on, NameServer cannot be empty")
	}

	dnsCfg := &DNS{
		Enable:       cfg.Enable,
		Listen:       cfg.Listen,
		IPv6:         cfg.IPv6,
		EnhancedMode: cfg.EnhancedMode,
	}
	var err error
	if dnsCfg.NameServer, err = parseNameServer(cfg.NameServer); err != nil {
		return nil, err
	}

	if len(cfg.DefaultNameserver) == 0 {
		return nil, errors.New("default nameserver should have at least one nameserver")
	}
	if dnsCfg.DefaultNameserver, err = parseNameServer(cfg.DefaultNameserver); err != nil {
		return nil, err
	}
	// check default nameserver is pure ip addr
	for _, ns := range dnsCfg.DefaultNameserver {
		host, _, err := net.SplitHostPort(ns.Addr)
		if err != nil || net.ParseIP(host) == nil {
			return nil, errors.New("default nameserver should be pure IP")
		}
	}

	if cfg.EnhancedMode == dns.FAKEIP {
		_, ipnet, err := net.ParseCIDR(cfg.FakeIPRange)
		if err != nil {
			return nil, err
		}

		var host *trie.DomainTrie
		// fake ip skip host filter
		if len(cfg.FakeIPFilter) != 0 {
			host = trie.New()
			for _, domain := range cfg.FakeIPFilter {
				if err := host.Insert(domain, true); err != nil {
					return nil, err
				}
			}
		}

		pool, err := fakeip.New(ipnet, 1000, host)
		if err != nil {
			return nil, err
		}

		dnsCfg.FakeIPRange = pool
	}

	if cfg.UseHosts {
		dnsCfg.Hosts = hosts
	}

	return dnsCfg, nil
}
