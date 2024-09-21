package avalanchecore

import (
	"avalanchecore/gen/proto/github.com/sqrtofpisqaured/avalanche/avalanchecore"
	"fmt"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"google.golang.org/protobuf/proto"
	"net"
)

type LinkQuality struct {
	// TBD
}

func SendAnnouncement(details avalancheClient, addr *net.UDPAddr) error {
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		// TODO
	}
	defer conn.Close()

	if addr.IP.To4() != nil {
		p := ipv4.NewPacketConn(conn)
		if err := p.SetTTL(1); err != nil {
			// TODO
		}
	} else {
		p := ipv6.NewPacketConn(conn)
		if err := p.SetHopLimit(1); err != nil {
			// TODO
		}
	}

	announceData := avalanchecore.AvalancheClient{
		Version:      1,
		ClientId:     details.ClientID.String(),
		Destination:  details.Destination.String(),
		Capabilities: []*avalanchecore.Capability{},
	}
	announceBytes, err := proto.Marshal(&announceData)
	if err != nil {
		return fmt.Errorf("Failed to serialize client announcement message: %v\n", err)
	}
	n, err := conn.Write(announceBytes)
	if err != nil {
		return fmt.Errorf("Failed to deliver client announcement message: %v\n", err)
	}
	if n != len(announceBytes) {
		return fmt.Errorf("Incomplete client announcement message delivered %d/%d bytes sent\n", n, len(announceBytes))
	}

	return nil
}
