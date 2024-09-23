package avalanchecore

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sqrtofpisquared/avalanche/avalanchecore/gen/proto/github.com/sqrtofpisqaured/avalanche/avalanchecore"
	"math/rand"
	"net"
	"time"
)

type ClientCapability struct {
}

type remoteStream struct {
	// Entry in stream table
}

type avalancheClient struct {
	ClientID     uuid.UUID
	Destination  net.UDPAddr
	Capabilities []ClientCapability
	ASPVersion   uint8
}

type remoteClient struct {
	avalancheClient
	LastSeenTimestamp uint64
	Quality           LinkQuality
}

type LocalClient struct {
	avalancheClient
	ClientTable map[uuid.UUID]remoteClient
	StreamTable map[uint16]remoteStream
}

func InitializeClient(cmnAddress string) (LocalClient, error) {
	var c LocalClient
	c.ClientTable = make(map[uuid.UUID]remoteClient)

	fmt.Println("Attempting connection to CMN...")
	cmn, err := cmnConnect(cmnAddress)
	if err != nil {
		return c, fmt.Errorf("Could not establish CMN connection: %v\n", err)
	}

	c.Destination = *cmn.LocalAddr
	c.ClientID = uuid.New()

	errors := make(chan error)
	go c.handleMessage(cmn.MessagesReceived, errors, &cmn)

	// TODO get capabilities of client (maybe pass them in?)

	msg := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_Announce{
			Announce: &avalanchecore.AvalancheClient{
				Version:      1, // TODO central place to get the version?
				ClientId:     c.ClientID.String(),
				Destination:  c.Destination.String(),
				Capabilities: []*avalanchecore.Capability{},
			},
		},
	}

	if err := cmn.broadcast(&msg); err != nil {
		return c, fmt.Errorf("Announcement message failed to deliver: %v\n", err)
	}

	return c, nil
}

func (client LocalClient) handleMessage(messages <-chan *avalanchecore.CMNMessage, errors chan<- error, cmn *ClientManagementNetwork) {
	for {
		m := <-messages

		switch m.Message.(type) {
		case *avalanchecore.CMNMessage_Announce:
			ann := m.GetAnnounce()
			go func() {
				err := client.handleAnnounce(ann, m.IsBroadcast, cmn)
				if err != nil {
					errors <- fmt.Errorf("Failed to handle announcement %v\n", err)
				}
			}()
		}
	}
}

func (client LocalClient) handleAnnounce(ann *avalanchecore.AvalancheClient, isBroadcast bool, cmn *ClientManagementNetwork) error {
	clientId, err := uuid.Parse(ann.ClientId)
	if err != nil {
		return fmt.Errorf("Invalid client ID received in announcment: %v\n", ann.ClientId)
	}

	addr, err := net.ResolveUDPAddr("udp", ann.Destination)
	if err != nil {
		return fmt.Errorf("Invalid destination address recieved in announcement %v\n", addr)
	}

	if client.Destination.String() == addr.String() {
		return nil
	}

	fmt.Printf("Updating client table for client %v at %v\n", clientId, addr)

	client.ClientTable[clientId] = remoteClient{
		avalancheClient: avalancheClient{
			ClientID:     clientId,
			Destination:  *addr,
			Capabilities: []ClientCapability{},
			ASPVersion:   uint8(ann.Version),
		},
		LastSeenTimestamp: cmn.getSyncedTime(),
		Quality:           LinkQuality{},
	}

	if isBroadcast {
		delay := rand.Intn(50)
		time.Sleep(time.Duration(delay) * time.Millisecond)

		// Construct reply to announcement
		msg := avalanchecore.CMNMessage{
			Message: &avalanchecore.CMNMessage_Announce{
				Announce: &avalanchecore.AvalancheClient{
					Version:      1, // TODO central place to get the version?
					ClientId:     client.ClientID.String(),
					Destination:  client.Destination.String(),
					Capabilities: []*avalanchecore.Capability{},
				},
			},
		}
		err = cmn.send(&msg, client.ClientTable[clientId].avalancheClient)
		if err != nil {
		}
	}

	return nil
}

func (client LocalClient) InitStream() AvalancheStream {
	var s AvalancheStream

	// Communicate with CMN to agree on a new AvalancheStream ID

	// Reference client table to find destination client

	// Negotiate connection with destination client over CMN

	// Announce stream to CMN

	// Perform AvalancheStream handshake

	return s
}
