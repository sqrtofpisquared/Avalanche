package avalanchecore

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
)

type AvalancheReceiver struct {
	listen        *net.UDPAddr
	channels      map[uint16]chan ClientPacket
	errors        chan error
	connection    *net.UDPConn
	cancelFunc    context.CancelFunc
	maxPacketSize int
}

func StartReceiver(maxPacketSize int) (*AvalancheReceiver, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0})
	if err != nil {
		return nil, err
	}

	errorChan := make(chan error, 1024)
	ctx, cancel := context.WithCancel(context.Background())

	s := AvalancheReceiver{
		conn.LocalAddr().(*net.UDPAddr),
		make(map[uint16]chan ClientPacket),
		errorChan,
		conn,
		cancel,
		maxPacketSize,
	}

	go s.receiveWorker(ctx)

	return &s, nil
}

func (s *AvalancheReceiver) receiveWorker(ctx context.Context) {
	buf := make([]byte, s.maxPacketSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, _, err := s.connection.ReadFromUDP(buf)
			if err != nil {
				s.errors <- fmt.Errorf("Failed to read from UDP: %v\n", err)
			}

			var dp DataPacket
			decoder := gob.NewDecoder(bytes.NewReader(buf[:n]))
			err = decoder.Decode(&dp)
			if err != nil {
				s.errors <- fmt.Errorf("Failed to deserialize packet: %v\n", err)
				continue
			}

			if dp.Version != AvalancheVersion {
				s.errors <- fmt.Errorf("Packet version mismatch recieved on stream %d\n", dp.StreamId)
			}

			channel, found := s.channels[dp.StreamId]
			if !found {
				s.errors <- fmt.Errorf("received packet for unknown stream %d\n", dp.StreamId)
			}

			channel <- ClientPacket{
				Flags:       dp.Flags,
				Payload:     dp.Payload,
				DesiredTime: dp.Timestamp,
			}
		}
	}
}

func (s *AvalancheReceiver) NextStreamID() (uint16, error) {
	sCount := len(s.channels)
	if sCount >= 65535 {
		return 0, errors.New("cannot get next stream ID - maximum amount of streams reached")
	}

	return uint16(sCount), nil
}

func (s *AvalancheReceiver) RegisterStream(streamID uint16) error {
	_, found := s.channels[streamID]
	if found {
		return errors.New("cannot register stream - already exists")
	}

	s.channels[streamID] = make(chan ClientPacket, 1024)

	return nil
}

func (s *AvalancheReceiver) ListenAddr() *net.UDPAddr {
	return s.listen
}

func (s *AvalancheReceiver) Data(streamID uint16) (<-chan ClientPacket, error) {
	data, found := s.channels[streamID]
	if !found {
		return nil, errors.New("unknown stream ID")
	}

	return data, nil
}

func (s *AvalancheReceiver) Errors() <-chan error {
	return s.errors
}

func (s *AvalancheReceiver) Close() {
	s.cancelFunc()
	close(s.errors)
	for _, v := range s.channels {
		close(v)
	}
}
