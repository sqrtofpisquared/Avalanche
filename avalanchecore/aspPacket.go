package avalanchecore

const (
	P_DATA int = 1
	P_FIN  int = 2
	P_ACK  int = 3
)

type CommonHeader struct {
	Version    uint8
	PacketType uint8
	StreamId   uint16
	Sequence   uint64
	Timestamp  uint64
}

type DataPacket struct {
	CommonHeader
	Flags         byte
	Reserved      [3]byte
	PayloadLength uint32
	Payload       []byte
}

type FinPacket struct {
	CommonHeader
	FinalSequenceNumber uint64
	ReasonCode          uint32
}

type AckPacket struct {
	CommonHeader
	AckSequenceStart uint64
	AckBitmap        []uint64
}
