package avalanchecore

import "fmt"

type DeliveryError struct {
	PacketID    uint64
	Description string
	Err         error
}

func (d *DeliveryError) Error() string {
	return fmt.Sprintf("(%d): %s - %v", d.PacketID, d.Description, d.Err)
}
