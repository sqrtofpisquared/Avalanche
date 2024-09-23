package avalanchecore

import (
	"avalanchecore/gen/proto/github.com/sqrtofpisqaured/avalanche/avalanchecore"
	"github.com/google/uuid"
	"net"
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

type localClient struct {
	avalancheClient
	ClientTable map[uuid.UUID]remoteClient
	StreamTable map[uint16]remoteStream
}

func InitializeClient(cmnAddress string) {
	var c localClient

	cmn := cmnConnect(cmnAddress)
	c.Destination = *cmn.LocalAddr
	c.ClientID = uuid.New()

	go HandleMessage(cmn.MessagesReceived)

	// TODO get capabilities of client (maybe pass them in?)

	msg := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_Announce{
			Announce: &avalanchecore.AvalancheClient{
				Version:      1,
				ClientId:     c.ClientID.String(),
				Destination:  c.Destination.String(),
				Capabilities: []*avalanchecore.Capability{},
			},
		},
	}

	if err := cmn.broadcast(&msg); err != nil {
		// TODO handle announcement failure
	}
}

func HandleMessage(messages chan avalanchecore.CMNMessage) {
	for {
		m := <-messages

		switch m.Message.(type) {
		case *avalanchecore.CMNMessage_Announce:
			// TODO update client table
		}
	}
}

func (client localClient) InitStream() AvalancheStream {
	var s AvalancheStream

	// Communicate with CMN to agree on a new AvalancheStream ID

	// Reference client table to find destination client

	// Negotiate connection with destination client over CMN

	// Announce stream to CMN

	// Perform AvalancheStream handshake

	return s
}
