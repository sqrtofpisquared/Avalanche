package main

import (
	"bufio"
	"fmt"
	"github.com/sqrtofpisquared/avalanche/avalanchecore"
	"os"
)

func main() {
	client, err := avalanchecore.InitializeClient("239.0.0.12:5515")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	fmt.Printf("Waiting on client %v\n", client.ClientID)

	fmt.Printf("Press any key to exit\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
