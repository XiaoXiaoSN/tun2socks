package tun

import (
	"fmt"

	"github.com/xjasonlyu/tun2socks/device"
	"github.com/xjasonlyu/tun2socks/device/rwbased"

	"github.com/songgao/water"
)

const defaultMTU = 1500

type TUN struct {
	*rwbased.Endpoint

	ifce *water.Interface
	mtu  uint32
	name string

	// windows only
	componentID string
	network     string
}

func Open(opts ...Option) (device.Device, error) {
	t := &TUN{}

	for _, opt := range opts {
		opt(t)
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID:   t.componentID,
			InterfaceName: t.name,
			Network:       t.network,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}
	t.ifce = ifce

	if t.mtu == 0 {
		t.mtu = defaultMTU
	}

	ep, err := rwbased.New(ifce, t.mtu)
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	t.Endpoint = ep

	return t, nil
}

func (t *TUN) Name() string {
	return t.name
}

func (t *TUN) Close() error {
	return t.ifce.Close()
}
