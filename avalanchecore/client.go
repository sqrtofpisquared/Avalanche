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
	TimeOffsets  *RollingAvg
	Quality      LinkQuality
}

type LocalClient struct {
	avalancheClient
	timeOffset    int64
	clientTableMu sync.RWMutex
	ClientTable   map[uuid.UUID]*remoteClient
	StreamTable   map[uint16]remoteStream
	Receiver      *AvalancheReceiver
}

func InitializeClient(cmn *ClientManagementNetwork) (*LocalClient, error) {
	var c LocalClient
	c.ClientTable = make(map[uuid.UUID]*remoteClient)
	c.StreamTable = make(map[uint16]remoteStream)
	c.Destination = *cmn.LocalAddr
	c.ClientID = uuid.New()
	r, err := StartReceiver(1024)
	if err != nil {
		return nil, fmt.Errorf("Could not set up stream receiver: %v\n", err)
	}
	c.Receiver = r

	go c.handleMessage(cmn)

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
		return nil, fmt.Errorf("Announcement message failed to deliver: %v\n", err)
	}

	go c.presence(cmn)

	return &c, nil
}

func (client *LocalClient) StartStream(toClient uuid.UUID, cmn *ClientManagementNetwork) error {
	client.clientTableMu.RLock()
	target, ok := client.ClientTable[toClient]
	client.clientTableMu.RUnlock()

	if !ok {
		return fmt.Errorf("Unknown client %v\n", toClient)
	}

	msg := &avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_StreamRequest{
			StreamRequest: &avalanchecore.StreamRequest{
				Version:    AvalancheVersion,
				ClientId:   client.ClientID.String(),
				TargetId:   target.ClientID.String(),
				StreamType: "TODO",
				Parameters: map[string]string{
					// TODO get stream parameters
				},
			},
		},
	}

	err := cmn.send(msg, &target.Destination)
	if err != nil {
		return fmt.Errorf("Failed to send stream request to client %v\n", toClient)
	}

	// TODO wait to receive reply - up to 10 seconds

	return nil
}

func (client *LocalClient) presence(cmn *ClientManagementNetwork) {
	for {
		time.Sleep(1 * time.Second)
		msg := avalanchecore.CMNMessage{
			Message: &avalanchecore.CMNMessage_Presence{
				Presence: &avalanchecore.Presence{
					Version:     AvalancheVersion,
					ClientId:    client.ClientID.String(),
					Destination: client.Destination.String(),
					Timestamp:   client.syncedTime(),
				},
			},
		}
		if err := cmn.broadcast(&msg); err != nil {
			fmt.Printf("Failed to send presence notification: %v\n", err)
			continue
		}

		fmt.Printf("Current offset: %d\n", client.timeOffset)

		// Clean up dead connections
		timeoutDuration := 1 * time.Minute
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

func (client *LocalClient) handleMessage(cmn *ClientManagementNetwork) {
	for {
		m := <-cmn.MessagesReceived

		switch m.Message.(type) {
		case *avalanchecore.CMNMessage_Announce:
			ann := m.GetAnnounce()
			go func() {
				err := client.handleAnnounce(ann, cmn)
				if err != nil {
					fmt.Printf("Failed to handle announcement %v\n", err)
				}
			}()
		case *avalanchecore.CMNMessage_AnnounceReply:
			aRep := m.GetAnnounceReply()
			err := client.handleAnnounceReply(aRep)
			if err != nil {
				fmt.Printf("Could not handle announcement reply: %v\n", err)
				continue
			}
		case *avalanchecore.CMNMessage_Presence:
			presence := m.GetPresence()
			err := client.handlePresence(presence, cmn)
			if err != nil {
				fmt.Printf("Could not handle presence message: %v\n", err)
				continue
			}
		}
	}
}

func (client *LocalClient) handleAnnounce(ann *avalanchecore.AvalancheClient, cmn *ClientManagementNetwork) error {
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
		TimeOffsets:  NewRollingAvg(10),
		LastSeenTime: time.Now(),
		Quality:      LinkQuality{},
	}
	client.clientTableMu.Unlock()

	// Delay for a random period of time between 0-50ms to avoid all clients responding simultaneously
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
	err = cmn.send(&msg, addr)
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
		TimeOffsets:  NewRollingAvg(10),
		Quality:      LinkQuality{},
	}
	client.clientTableMu.Unlock()

	return nil
}

func (client *LocalClient) handlePresence(p *avalanchecore.Presence, cmn *ClientManagementNetwork) error {
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
		err = cmn.send(&msg, addr)
		if err != nil {
			return fmt.Errorf("Could not announce to client: %v\n", err)
		}
		return nil
	} else {
		client.clientTableMu.Lock()
		client.ClientTable[clientId].LastSeenTime = time.Now()
		o := p.Timestamp - client.syncedTime()
		client.ClientTable[clientId].TimeOffsets.Push(o)

		client.clientTableMu.Unlock()

		// Compute the new average offset
		client.clientTableMu.RLock()
		var allOffset int64
		for _, v := range client.ClientTable {
			allOffset += v.TimeOffsets.Avg()
		}
		client.timeOffset = allOffset / int64(len(client.ClientTable))
		client.clientTableMu.RUnlock()
	}

	return nil
}

func (client *LocalClient) handleStreamRequest(req *avalanchecore.StreamRequest, cmn *ClientManagementNetwork) error {
	requesterId, err := uuid.Parse(req.ClientId)
	if err != nil {
		return fmt.Errorf("Invalid client id received in stream request\n")
	}

	receiverId, err := uuid.Parse(req.TargetId)
	if err != nil || receiverId != client.ClientID {
		return fmt.Errorf("Invalid target id received in stream request\n")
	}

	client.clientTableMu.RLock()
	requester, ok := client.ClientTable[requesterId]
	client.clientTableMu.RUnlock()
	if !ok {
		return fmt.Errorf("Received stream request from unknown client %v\n", requesterId)
	}

	// TODO evaluate if stream type and parameters are supported

	// Create a new stream object & begin listening
	// Stream object will handle ASP stream handshake and everything stream-specific

	accept := avalanchecore.CMNMessage{
		Message: &avalanchecore.CMNMessage_StreamRequestReply{
			StreamRequestReply: &avalanchecore.StreamRequestReply{
				ClientId:    client.ClientID.String(),
				TargetId:    requester.ClientID.String(),
				ListeningOn: "",
				Accept:      true,
			},
		},
	}

	err = cmn.send(&accept, &requester.Destination)
	if err != nil {
		return fmt.Errorf("Failed to send stream request reply to %v\n", requesterId)
	}

	return nil
}

func (client *LocalClient) syncedTime() int64 {
	systemTime := time.Now()
	return systemTime.UnixNano() + client.timeOffset
}
