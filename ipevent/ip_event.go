package ipevent

import (
	"context"
	"ip-blackcage/event"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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
	r.ipchain <- event.NewEventData(time.Now().UnixMilli(), ip.SrcIP.String())
}

func (r *ipEventReader) Open(ctx context.Context) (<-chan event.IEventData, error) {
	return r.ipchain, nil
}
