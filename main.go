package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)


func main() {
	prog := os.Args[0]
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Println("Usage:", prog, "[Party ID] [Input]")
		os.Exit(1)
	}

	partyID, errPartyID := strconv.ParseUint(args[0], 10, 64)
	if errPartyID != nil {
		fmt.Println("Party ID should be an unsigned integer")
		os.Exit(1)
	}

	partyInput, errPartyInput := strconv.ParseUint(args[1], 10, 64)
	if errPartyInput != nil {
		fmt.Println("Party input should be an unsigned integer")
		os.Exit(1)
	}

	Client(PartyID(partyID), partyInput)
}


func Client(partyID PartyID, partyInput uint64) {

	//N := uint64(len(peers))
	peers := map[PartyID]string {
		0: "localhost:6660",
		1: "localhost:6661",
		2: "localhost:6662",
	}

	// Create a local party 
	lp, err := NewLocalParty(partyID, peers)
	check(err)

	// Create the network for the circuit
	network, err := NewTCPNetwork(lp)
	check(err)

	// Connect the circuit network 
	err = network.Connect(lp)
	check(err)
	fmt.Println(lp, "connected")
	<- time.After(time.Second) // Leave time for others to connect

	// Create a new circuit evaluation protocol 
	dummyProtocol := lp.NewDummyProtocol(partyInput)
	// Bind evaluation protocol to the network
	dummyProtocol.BindNetwork(network)

	// Evaluate the circuit
	dummyProtocol.Run()

	fmt.Println(lp, "completed with output", dummyProtocol.Output)
}
