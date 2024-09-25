package avalanchecore

import "net"

type AvalancheStream struct {
	StreamId   uint16
	Sequence   uint64
	ListenAddr *net.UDPAddr
}

func StartStream() *AvalancheStream {

	return nil
}
