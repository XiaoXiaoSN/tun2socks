package tun

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/xjasonlyu/tun2socks/device"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	"gvisor.dev/gvisor/pkg/tcpip/link/rawfile"
	"gvisor.dev/gvisor/pkg/tcpip/link/tun"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type TUN struct {
	stack.LinkEndpoint

	fd   int
	mtu  uint32
	name string
}

func Open(opts ...Option) (device.Device, error) {
	t := &TUN{}

	for _, opt := range opts {
		opt(t)
	}

	if len(t.name) >= unix.IFNAMSIZ {
		return nil, errors.New("interface name too long")
	}

	fd, err := tun.Open(t.name)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}
	t.fd = fd

	if t.mtu > 0 {
		if err := setMTU(t.name, t.mtu); err != nil {
			return nil, fmt.Errorf("set mtu: %w", err)
		}
	}

	mtu, err := rawfile.GetMTU(t.name)
	if err != nil {
		return nil, fmt.Errorf("get mtu: %w", err)
	}
	t.mtu = mtu

	ep, err := fdbased.New(&fdbased.Options{
		MTU: t.mtu,
		FDs: []int{
			t.fd,
		},
		// TUN only
		EthernetHeader: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	t.LinkEndpoint = ep

	return t, nil
}

func (t *TUN) Name() string {
	return t.name
}

func (t *TUN) Close() error {
	return unix.Close(t.fd)
}

func setMTU(name string, n uint32) error {
	// open datagram socket
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)
	if err != nil {
		return err
	}

	defer unix.Close(fd)

	const ifReqSize = unix.IFNAMSIZ + 64

	// do ioctl call
	var ifr [ifReqSize]byte
	copy(ifr[:], name)
	*(*uint32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = n
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCSIFMTU),
		uintptr(unsafe.Pointer(&ifr[0])),
	)

	if errno != 0 {
		return errors.New("failed to set MTU")
	}

	return nil
}