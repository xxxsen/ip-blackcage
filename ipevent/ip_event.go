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
	c := applyOpts(opts...)
	r := &ipEventReader{c: c, ipchain: make(chan event.IEventData, 1024)}
	handler, err := pcap.OpenLive(r.c.iface, 1600, true, pcap.BlockForever)
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

func (r *ipEventReader) decodeNetInfo(packet gopacket.Packet) (string, string, uint16, uint16, bool) {
	var srcip, dstip gopacket.Endpoint
	var srcport, dstport uint16
	extractFn := func() bool {
		nl := packet.NetworkLayer()
		if nl == nil {
			return false
		}
		if nl.LayerType() == layers.LayerTypeIPv6 { // 先不处理ipv6
			return false
		}
		srcip, dstip = nl.NetworkFlow().Endpoints()
		tpl := packet.TransportLayer()
		if tpl == nil {
			return false
		}

		if ly, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP); ok {
			if !ly.SYN {
				return false
			}
			srcport = uint16(ly.SrcPort)
			dstport = uint16(ly.DstPort)
			return true
		}
		if ly, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP); ok {
			srcport = uint16(ly.SrcPort)
			dstport = uint16(ly.DstPort)
			return true
		}
		return false
	}
	if !extractFn() {
		return "", "", 0, 0, false
	}
	return srcip.String(), dstip.String(), srcport, dstport, true
}

func (r *ipEventReader) handlePacket(packet gopacket.Packet) {
	srcip, dstip, srcport, dstport, ok := r.decodeNetInfo(packet)
	if !ok {
		return
	}
	if _, ok := r.c.exitIps[srcip]; ok {
		return
	}
	if _, ok := r.c.portMap[uint16(dstport)]; !ok {
		return
	}
	logutil.GetLogger(context.Background()).Debug("recv port scan request",
		zap.String("src", fmt.Sprintf("%s:%d", srcip, srcport)),
		zap.String("dst", fmt.Sprintf("%s:%d", dstip, dstport)),
	)
	r.ipchain <- event.NewEventData(
		string(event.EventTypePortScan),
		time.Now().UnixMilli(),
		&IPEventData{
			SrcIP:   srcip,
			DstIP:   dstip,
			SrcPort: srcport,
			DstPort: dstport,
		},
	)
}

func (r *ipEventReader) Open(ctx context.Context) (<-chan event.IEventData, error) {
	return r.ipchain, nil
}
