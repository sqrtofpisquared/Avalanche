package main

import (
	"bufio"
	"fmt"
	"github.com/sqrtofpisquared/avalanche/avalanchecore"
	"os"
)

func main() {
	fmt.Println("Attempting connection to CMN...")
	cmnAddress := "239.0.0.12:5515"
	cmn, err := avalanchecore.CMNConnect(cmnAddress)
	if err != nil {
		fmt.Printf("Could not establish CMN connection to %v: %v\n", cmnAddress, err)
	}

	client, err := avalanchecore.InitializeClient(cmn)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	fmt.Printf("Waiting on client %v\n", client.ClientID)

	fmt.Printf("Press any key to exit\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
