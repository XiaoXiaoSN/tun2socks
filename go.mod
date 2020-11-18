module github.com/xjasonlyu/tun2socks

go 1.15

require (
	github.com/Dreamacro/clash v0.0.0
	github.com/Dreamacro/go-shadowsocks2 v0.1.6
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-chi/render v1.0.1
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/miekg/dns v1.1.35 // indirect
	github.com/oschwald/maxminddb-golang v1.7.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/stretchr/testify v1.6.1
	go.uber.org/atomic v1.7.0
	golang.org/x/crypto v0.0.0-20201117144127-c1f2f97bffc9 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20201117222635-ba5294a509c7
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.zx2c4.com/wireguard v0.0.20200320
	gvisor.dev/gvisor v0.0.0-20201114021409-8033d1baf17c
)

replace github.com/Dreamacro/clash => github.com/xjasonlyu/clash v0.15.1-0.20201118021831-555be37818d3
