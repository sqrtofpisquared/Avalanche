package avalanchecore

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
)

type AvalancheSender struct {
	StreamId      uint16
	data          chan ClientPacket
	ToAddr        *net.UDPAddr
	sequence      uint64
	sequenceLock  sync.Mutex
	maxPacketSize int
	errors        chan error
	connection    *net.UDPConn
	cancelFunc    context.CancelFunc
}

type ClientPacket struct {
	Flags       byte
	Payload     []byte
	DesiredTime uint64
}

func StartSender(id uint16, to *net.UDPAddr, maxPacketSize int) (*AvalancheSender, error) {
	conn, err := net.DialUDP("udp", nil, to)
	if err != nil {
		return nil, err
	}

	data := make(chan ClientPacket, 1024)
	errorChan := make(chan error, 1024)
	ctx, cancel := context.WithCancel(context.Background())

	s := AvalancheSender{
		id,
		data,
		to,
		0,
		sync.Mutex{},
		maxPacketSize,
		errorChan,
		conn,
		cancel,
	}

	go s.sendWorker(ctx)

	return &s, nil
}

func (s *AvalancheSender) nextHeader(pType int) CommonHeader {
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

func (s *AvalancheSender) sendWorker(ctx context.Context) {
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

func (s *AvalancheSender) Close() {
	s.cancelFunc()
	close(s.errors)
	close(s.data)
}

func (s *AvalancheSender) Data() chan<- ClientPacket {
	return s.data
}
