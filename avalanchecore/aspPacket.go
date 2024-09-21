package avalanchecore

const (
	P_DATA int = 1
	P_INIT int = 2
	P_FIN  int = 3
	P_ACK  int = 4
	P_NACK int = 5
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

type InitPacket struct {
	CommonHeader
	StreamType       uint32
	InitSequence     uint64
	ParamBlockLength uint32
	ParamBlock       []byte
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

type NackPacket struct {
	CommonHeader
	NackSequenceStart uint64
	NackBitmap        []uint64
}
