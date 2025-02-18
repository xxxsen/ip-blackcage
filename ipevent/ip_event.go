package ipevent

import (
	"context"
	"fmt"
	"ip-blackcage/event"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type ipEventReader struct {
	c       *config
	ipchain chan event.IEventData
}

func NewIPEventReader(opts ...Option) (event.IEventReader, error) {
	c := &config{
		interface_: "eth0",
		portm:      make(map[uint16]struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	r := &ipEventReader{c: c, ipchain: make(chan event.IEventData, 1024)}
	handler, err := pcap.OpenLive(r.c.interface_, 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	go r.start(handler)
	return r, nil
}

func (r *ipEventReader) start(handler *pcap.Handle) {
	defer handler.Close()
	packetSource := gopacket.NewPacketSource(handler, handler.LinkType())
	for packet := range packetSource.Packets() {
		r.handlePacket(packet)
	}
}

func (r *ipEventReader) handlePacket(packet gopacket.Packet) {
	// 获取IP层和TCP层
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}

	ip, _ := ipLayer.(*layers.IPv4)
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}

	tcp, _ := tcpLayer.(*layers.TCP)

	if _, ok := r.c.portm[uint16(tcp.DstPort)]; !ok {
		return
	}
	logutil.GetLogger(context.Background()).Debug("recv port scan request",
		zap.String("src", fmt.Sprintf("%s:%d", ip.SrcIP.String(), tcp.SrcPort)),
		zap.String("dst", fmt.Sprintf("%s:%d", ip.DstIP.String(), tcp.DstPort)),
	)
	r.ipchain <- event.NewEventData(
		string(event.EventTypePortScan),
		time.Now().UnixMilli(),
		&IPEventData{
			SrcIP:   ip.SrcIP.String(),
			DstIP:   ip.DstIP.String(),
			SrcPort: uint16(tcp.SrcPort),
			DstPort: uint16(tcp.DstPort),
		},
	)
}

func (r *ipEventReader) Open(ctx context.Context) (<-chan event.IEventData, error) {
	return r.ipchain, nil
}
