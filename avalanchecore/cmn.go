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

type ClientManagementNetwork struct {
	ListenAddr *net.UDPAddr
	Conn       *net.UDPConn
}

func cmnConnect(address string) ClientManagementNetwork {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		// TODO
	}
	if addr == nil {
		// TODO handle missing address
	}

	cmn := ClientManagementNetwork{ListenAddr: addr}
	go cmn.cmnListen()

	return cmn
}

func (cmn *ClientManagementNetwork) sendAnnouncement(details avalancheClient) error {
	if cmn.Conn == nil {
		return fmt.Errorf("Cannot send announcement as client management network is currently disconnected\n")
	}

	message := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_Announce{
			Announce: &avalanchecore.AvalancheClient{
				Version:      1,
				ClientId:     details.ClientID.String(),
				Destination:  details.Destination.String(),
				Capabilities: []*avalanchecore.Capability{},
			},
		},
	}
	announceBytes, err := proto.Marshal(&message)
	if err != nil {
		return fmt.Errorf("Failed to serialize client announcement message: %v\n", err)
	}

	n, err := cmn.Conn.Write(announceBytes)
	if err != nil {
		return fmt.Errorf("Failed to deliver client announcement message: %v\n", err)
	}
	if n != len(announceBytes) {
		return fmt.Errorf("Incomplete client announcement message delivered %d/%d bytes sent\n", n, len(announceBytes))
	}

	return nil
}

func (cmn *ClientManagementNetwork) cmnListen() {
	// TODO explicitly handle ipv4/ipv6 multicast listener
	conn, err := net.ListenMulticastUDP("udp", nil, cmn.ListenAddr)
	if err != nil {
		// TODO
	}
	defer cmn.Conn.Close()

	if cmn.ListenAddr.IP.To4() != nil {
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

	cmn.Conn = conn

	var errors chan<- error

	buffer := make([]byte, 1024)
	for {
		n, source, err := cmn.Conn.ReadFromUDP(buffer)
		if err != nil {
			// TODO
			continue
		}
		data := make([]byte, n)
		copy(data, buffer[:n])
		go cmn.handleReceivePacket(source, data, errors)
	}
}

func (cmn *ClientManagementNetwork) handleReceivePacket(source *net.UDPAddr, data []byte, eChan chan<- error) {
	var m avalanchecore.CMNMessage
	err := proto.Unmarshal(data, &m)
	if err != nil {
		eChan <- fmt.Errorf("Failed to unmarshal message from %v: %v\n", source, err)
	}

	switch m.Message.(type) {
	case *avalanchecore.CMNMessage_Announce:
		// TODO update client table
		return

	default:
		eChan <- fmt.Errorf("Unknown message type from %v\n", source)
	}
}
