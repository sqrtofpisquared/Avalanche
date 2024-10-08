package avalanchecore

import (
	"fmt"
	"github.com/sqrtofpisquared/avalanche/avalanchecore/gen/proto/github.com/sqrtofpisqaured/avalanche/avalanchecore"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"google.golang.org/protobuf/proto"
	"net"
)

type LinkQuality struct {
	// TBD
}

type ClientManagementNetwork struct {
	BroadcastAddr    *net.UDPAddr
	BroadcastConn    *net.UDPConn
	LocalAddr        *net.UDPAddr
	LocalConn        *net.UDPConn
	MessagesReceived chan *avalanchecore.CMNMessage
}

func CMNConnect(address string) (*ClientManagementNetwork, error) {
	bAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve CMN multicast address\n")
	}

	bConn, err := net.ListenMulticastUDP("udp", nil, bAddr)
	if err != nil {
		return nil, fmt.Errorf("Could not listen on broadcast address: %v\n", err)
	}

	lConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0})
	if err != nil {
		return nil, fmt.Errorf("Could not listen on local address: %v\n", err)
	}

	mChannel := make(chan *avalanchecore.CMNMessage)
	cmn := ClientManagementNetwork{
		BroadcastAddr:    bAddr,
		BroadcastConn:    bConn,
		LocalAddr:        lConn.LocalAddr().(*net.UDPAddr),
		LocalConn:        lConn,
		MessagesReceived: mChannel,
	}
	go cmn.listen(cmn.LocalConn)
	go cmn.listen(cmn.BroadcastConn)

	return &cmn, nil
}

func (cmn *ClientManagementNetwork) broadcast(msg *avalanchecore.CMNMessage) error {
	conn, err := net.DialUDP("udp", nil, cmn.BroadcastAddr)
	if err != nil {
		return fmt.Errorf("Could not resolve broadcast address when sending message: %v\n", err)
	}
	defer conn.Close()

	if cmn.BroadcastAddr.IP.To4() != nil {
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

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed to serialize client message: %v\n", err)
	}

	n, err := conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("Failed to broadcast message to CMN: %v\n", err)
	}
	if n != len(msgBytes) {
		return fmt.Errorf("Incomplete message delivered to CMN - %d/%d bytes sent\n", n, len(msgBytes))
	}

	return nil
}

func (cmn *ClientManagementNetwork) send(msg *avalanchecore.CMNMessage, addr *net.UDPAddr) error {
	// Open a new UDP
	conn, err := net.DialUDP("udp", nil, addr) // TODO this should be TCP
	if err != nil {
		return fmt.Errorf("Failed to connect to %v: %v\n", addr, err)
	}
	defer conn.Close()

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed to marshal message\n")
	}

	n, err := conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("Failed to deliver message to %v: %v\n", addr, err)
	}
	if n != len(msgBytes) {
		return fmt.Errorf("Incopmlete message sent to %v - %d/%d bytes sent\n", addr, n, len(msgBytes))
	}

	return nil
}

func (cmn *ClientManagementNetwork) listen(conn *net.UDPConn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, source, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Error receiving message from %v: %v\n", source, err)
			continue
		}
		data := make([]byte, n)
		copy(data, buffer[:n])
		m := avalanchecore.CMNMessage{}
		err = proto.Unmarshal(data, &m)
		if err != nil {
			fmt.Printf("Failed to unmarshal message from %v: %v\n", source, err)
			continue
		}

		cmn.MessagesReceived <- &m
	}
}
