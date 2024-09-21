package avalanchecore

import "net"

type AvalancheStream struct {
	StreamId    uint16
	Sequence    uint64
	Destination net.Addr
}
