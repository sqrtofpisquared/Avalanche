package avalanchecore

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
)

type AvalancheStream struct {
	StreamId      uint16
	FromAddr      *net.UDPAddr
	ToAddr        *net.UDPAddr
	data          chan ClientPacket
	errors        chan error
	connection    *net.UDPConn
	sequence      uint64
	sequenceLock  sync.Mutex
	maxPacketSize int
	cancelFunc    context.CancelFunc
}

type ClientPacket struct {
	Flags       byte
	Payload     []byte
	DesiredTime uint64
}

func StartSender(id uint16, from *net.UDPAddr, to *net.UDPAddr, maxPacketSize int) (*AvalancheStream, error) {
	conn, err := net.DialUDP("udp", nil, to)
	if err != nil {
		return nil, err
	}

	data := make(chan ClientPacket, 1024)
	errorChan := make(chan error, 1024)
	ctx, cancel := context.WithCancel(context.Background())

	s := AvalancheStream{
		id,
		from,
		to,
		data,
		errorChan,
		conn,
		0,
		sync.Mutex{},
		maxPacketSize,
		cancel,
	}

	go s.sendWorker(ctx)

	return &s, nil
}

func StartReceiver(id uint16, listen *net.UDPAddr, maxPacketSize int) (*AvalancheStream, error) {
	conn, err := net.ListenUDP("udp", listen)
	if err != nil {
		return nil, err
	}

	data := make(chan ClientPacket, 1024)
	errorChan := make(chan error, 1024)
	ctx, cancel := context.WithCancel(context.Background())

	s := AvalancheStream{
		id,
		nil,
		listen,
		data,
		errorChan,
		conn,
		0,
		sync.Mutex{},
		maxPacketSize,
		cancel,
	}

	go s.receiveWorker(ctx)

	return &s, nil
}

func (s *AvalancheStream) nextHeader(pType int) CommonHeader {
	s.sequenceLock.Lock()
	s.sequence++

	h := CommonHeader{
		Version:    AvalancheVersion,
		PacketType: uint8(pType),
		Sequence:   s.sequence,
		StreamId:   s.StreamId,
	}
	s.sequenceLock.Unlock()

	return h
}

func (s *AvalancheStream) sendWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case nextPacket := <-s.data:
			dataLength := len(nextPacket.Payload)
			maxLength := s.maxPacketSize - HEADER_SIZE_BYTES - DATA_RESERVED_BYTES
			if dataLength+HEADER_SIZE_BYTES+DATA_RESERVED_BYTES > s.maxPacketSize {
				s.errors <- fmt.Errorf("Packet data length exceeds maximum allowed length of %d\n", maxLength)
				continue
			}

			dp := DataPacket{
				s.nextHeader(P_DATA),
				nextPacket.DesiredTime,
				nextPacket.Flags,
				[3]byte{},
				uint32(dataLength),
				nextPacket.Payload,
			}
			var encoded bytes.Buffer
			enc := gob.NewEncoder(&encoded)
			err := enc.Encode(dp)
			if err != nil {
				s.errors <- &DeliveryError{dp.CommonHeader.Sequence, "Failed to serialize packet", err}
				continue
			}

			b := encoded.Bytes()

			n, err := s.connection.Write(b)
			if err != nil {
				s.errors <- &DeliveryError{dp.CommonHeader.Sequence, "Could not send packet to receiver", err}
				continue
			}
			if n < len(b) {
				s.errors <- &DeliveryError{dp.CommonHeader.Sequence, "Partial write", err}
				continue
			}
		}
	}
}

func (s *AvalancheStream) receiveWorker(ctx context.Context) {
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

			packet := ClientPacket{
				Flags:       dp.Flags,
				Payload:     dp.Payload,
				DesiredTime: dp.Timestamp,
			}

			s.data <- packet
		}
	}
}

func (s *AvalancheStream) Close() {
	s.cancelFunc()
	close(s.data)
	close(s.errors)
}

func (s *AvalancheStream) Errors() <-chan error {
	return s.errors
}

func (s *AvalancheStream) Data() chan ClientPacket {
	return s.data
}
