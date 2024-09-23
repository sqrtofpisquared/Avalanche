package avalanchecore

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sqrtofpisquared/avalanche/avalanchecore/gen/proto/github.com/sqrtofpisqaured/avalanche/avalanchecore"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	AvalancheVersion = 1
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
	LastSeenTime time.Time
	Quality      LinkQuality
}

type LocalClient struct {
	avalancheClient
	cmn           *ClientManagementNetwork
	clientTableMu sync.RWMutex
	ClientTable   map[uuid.UUID]*remoteClient
	StreamTable   map[uint16]remoteStream
}

func InitializeClient(cmnAddress string) (LocalClient, error) {
	var c LocalClient
	c.ClientTable = make(map[uuid.UUID]*remoteClient)
	c.StreamTable = make(map[uint16]remoteStream)

	fmt.Println("Attempting connection to CMN...")
	cmn, err := cmnConnect(cmnAddress)
	if err != nil {
		return c, fmt.Errorf("Could not establish CMN connection: %v\n", err)
	}

	c.Destination = *cmn.LocalAddr
	c.ClientID = uuid.New()
	c.cmn = &cmn

	errors := make(chan error)
	go c.handleMessage(errors)

	go func() {
		for err := range errors {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	// TODO get capabilities of client (maybe pass them in?)

	msg := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_Announce{
			Announce: &avalanchecore.AvalancheClient{
				Version:      AvalancheVersion,
				ClientId:     c.ClientID.String(),
				Destination:  c.Destination.String(),
				Capabilities: []*avalanchecore.Capability{},
			},
		},
	}

	if err := cmn.broadcast(&msg); err != nil {
		return c, fmt.Errorf("Announcement message failed to deliver: %v\n", err)
	}

	go c.presence()

	return c, nil
}

func (client *LocalClient) presence() {
	for {
		time.Sleep(1 * time.Minute)
		msg := avalanchecore.CMNMessage{
			Message: &avalanchecore.CMNMessage_Presence{
				Presence: &avalanchecore.Presence{
					Version:     AvalancheVersion,
					ClientId:    client.ClientID.String(),
					Destination: client.Destination.String(),
				},
			},
		}
		if err := client.cmn.broadcast(&msg); err != nil {
			fmt.Printf("Failed to send presence notification: %v\n", err)
			continue
		}
		timeoutDuration := 10 * time.Minute
		var deadClients []uuid.UUID

		client.clientTableMu.RLock()
		for _, v := range client.ClientTable {
			if time.Now().Sub(v.LastSeenTime) > timeoutDuration {
				deadClients = append(deadClients, v.ClientID)
			}
		}
		client.clientTableMu.RUnlock()

		client.clientTableMu.Lock()
		for _, id := range deadClients {
			fmt.Printf("Client %v not seen for %v - removing from client table", id, timeoutDuration)
			delete(client.ClientTable, id)
		}
		client.clientTableMu.Unlock()
	}
}

func (client *LocalClient) handleMessage(errors chan<- error) {
	for {
		m := <-client.cmn.MessagesReceived

		switch m.Message.(type) {
		case *avalanchecore.CMNMessage_Announce:
			ann := m.GetAnnounce()
			go func() {
				err := client.handleAnnounce(ann)
				if err != nil {
					errors <- fmt.Errorf("Failed to handle announcement %v\n", err)
				}
			}()
		case *avalanchecore.CMNMessage_AnnounceReply:
			aRep := m.GetAnnounceReply()
			err := client.handleAnnounceReply(aRep)
			if err != nil {
				errors <- fmt.Errorf("Could not handle announcement reply: %v\n", err)
			}
		case *avalanchecore.CMNMessage_Presence:
			presence := m.GetPresence()
			err := client.handlePresence(presence)
			if err != nil {
				errors <- fmt.Errorf("Could not handle presence message: %v\n", err)
			}
		}
	}
}

func (client *LocalClient) handleAnnounce(ann *avalanchecore.AvalancheClient) error {
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

	fmt.Printf("New client %v at %v\n", clientId, addr)

	client.clientTableMu.Lock()
	client.ClientTable[clientId] = &remoteClient{
		avalancheClient: avalancheClient{
			ClientID:     clientId,
			Destination:  *addr,
			Capabilities: []ClientCapability{},
			ASPVersion:   uint8(ann.Version),
		},
		LastSeenTime: time.Now(),
		Quality:      LinkQuality{},
	}
	client.clientTableMu.Unlock()

	delay := rand.Intn(50)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Construct reply to announcement
	msg := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_AnnounceReply{
			AnnounceReply: &avalanchecore.AvalancheClient{
				Version:      AvalancheVersion,
				ClientId:     client.ClientID.String(),
				Destination:  client.Destination.String(),
				Capabilities: []*avalanchecore.Capability{},
			},
		},
	}
	err = client.cmn.send(&msg, addr)
	if err != nil {
		return fmt.Errorf("Could not send announcement to %v: %v\n", addr, err)
	}

	return nil
}

func (client *LocalClient) handleAnnounceReply(p *avalanchecore.AvalancheClient) error {
	clientId, err := uuid.Parse(p.ClientId)
	if err != nil {
		return fmt.Errorf("Invalid client ID received in announcment: %v\n", p.ClientId)
	}

	addr, err := net.ResolveUDPAddr("udp", p.Destination)
	if err != nil {
		return fmt.Errorf("Invalid destination address recieved in announcement %v\n", addr)
	}

	fmt.Printf("Client registered in response %v at %v\n", clientId, addr)

	client.clientTableMu.Lock()
	client.ClientTable[clientId] = &remoteClient{
		avalancheClient: avalancheClient{
			ClientID:     clientId,
			Destination:  *addr,
			Capabilities: []ClientCapability{},
			ASPVersion:   uint8(p.Version),
		},
		LastSeenTime: time.Now(),
		Quality:      LinkQuality{},
	}
	client.clientTableMu.Unlock()

	return nil
}

func (client *LocalClient) handlePresence(p *avalanchecore.Presence) error {
	clientId, err := uuid.Parse(p.ClientId)
	if err != nil {
		return fmt.Errorf("Invalid client ID received in announcment: %v\n", p.ClientId)
	}

	if clientId == client.ClientID {
		return nil
	}

	client.clientTableMu.RLock()
	_, ok := client.ClientTable[clientId]
	client.clientTableMu.RUnlock()

	if !ok {
		// Client unknown - announce presence to client
		addr, err := net.ResolveUDPAddr("udp", p.Destination)
		if err != nil {
			return fmt.Errorf("Invalid destination address recieved in announcement %v\n", addr)
		}

		msg := avalanchecore.CMNMessage{
			Message: &avalanchecore.CMNMessage_Announce{
				Announce: &avalanchecore.AvalancheClient{
					Version:      AvalancheVersion,
					ClientId:     client.ClientID.String(),
					Destination:  client.Destination.String(),
					Capabilities: []*avalanchecore.Capability{},
				},
			},
		}
		err = client.cmn.send(&msg, addr)
		if err != nil {
			return fmt.Errorf("Could not announce to client: %v\n", err)
		}
		return nil
	} else {
		client.clientTableMu.Lock()
		client.ClientTable[clientId].LastSeenTime = time.Now()
		client.clientTableMu.Unlock()
	}

	return nil
}

func (client *LocalClient) InitStream() AvalancheStream {
	var s AvalancheStream

	// Communicate with CMN to agree on a new AvalancheStream ID

	// Reference client table to find destination client

	// Negotiate connection with destination client over CMN

	// Announce stream to CMN

	// Perform AvalancheStream handshake

	return s
}
