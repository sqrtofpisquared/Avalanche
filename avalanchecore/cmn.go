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
	BroadcastAddr    *net.UDPAddr
	LocalAddr        *net.UDPAddr
	Conn             *net.UDPConn
	MessagesReceived chan *avalanchecore.CMNMessage
}

func cmnConnect(address string) ClientManagementNetwork {
	bAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		// TODO
	}

	// Setup a new local address
	lAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		// TODO
	}

	mChannel := make(chan *avalanchecore.CMNMessage)
	cmn := ClientManagementNetwork{
		BroadcastAddr:    bAddr,
		LocalAddr:        lAddr,
		MessagesReceived: mChannel,
	}
	go cmn.listenLocal()
	go cmn.listenBroadcast()

	return cmn
}

func (cmn *ClientManagementNetwork) broadcast(msg *avalanchecore.CMNMessage) error {
	if cmn.Conn == nil {
		return fmt.Errorf("Cannot send announcement as client management network is currently disconnected\n")
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed to serialize client announcement message: %v\n", err)
	}

	n, err := cmn.Conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("Failed to broadcast message to CMN: %v\n", err)
	}
	if n != len(msgBytes) {
		return fmt.Errorf("Incomplete message delivered to CMN - %d/%d bytes sent\n", n, len(msgBytes))
	}

	return nil
}

func (cmn *ClientManagementNetwork) send(msg *avalanchecore.CMNMessage, client avalancheClient) error {
	// Open a new UDP
	conn, err := net.DialUDP("udp", nil, &client.Destination)
	if err != nil {
		return fmt.Errorf("Failed to connect to client %v at %v\n", client.ClientID, client.Destination)
	}
	defer conn.Close()

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed to marshal message\n")
	}

	n, err := conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("Failed to deliver message to %v: %v\n", client.Destination, err)
	}
	if n != len(msgBytes) {
		return fmt.Errorf("Incopmlete message sent to %v - %d/%d bytes sent\n", client.Destination, n, len(msgBytes))
	}

	return nil
}

func (cmn *ClientManagementNetwork) listenBroadcast() {
	// TODO explicitly handle ipv4/ipv6 multicast listener
	conn, err := net.ListenMulticastUDP("udp", nil, cmn.BroadcastAddr)
	if err != nil {
		// TODO
	}
	cmn.Conn = conn
	defer cmn.Conn.Close()

	if cmn.BroadcastAddr.IP.To4() != nil {
		p := ipv4.NewPacketConn(cmn.Conn)
		if err := p.SetTTL(1); err != nil {
			// TODO
		}
	} else {
		p := ipv6.NewPacketConn(cmn.Conn)
		if err := p.SetHopLimit(1); err != nil {
			// TODO
		}
	}

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
	var m *avalanchecore.CMNMessage
	err := proto.Unmarshal(data, m)
	if err != nil {
		eChan <- fmt.Errorf("Failed to unmarshal message from %v: %v\n", source, err)
	}

	cmn.MessagesReceived <- m
}

func (cmn *ClientManagementNetwork) listenLocal() {
	conn, err := net.ListenUDP("udp", cmn.LocalAddr)
	if err != nil {
		// TODO
	}
	defer conn.Close()

	var errors chan<- error
	buffer := make([]byte, 1024)
	for {
		n, source, err := conn.ReadFromUDP(buffer)
		if err != nil {
			// TODO
			continue
		}
		data := make([]byte, n)
		copy(data, buffer[:n])

		go cmn.handleReceivePacket(source, data, errors)
	}
}

func (cmn *ClientManagementNetwork) getSyncedTime() uint64 {
	return 0
}
