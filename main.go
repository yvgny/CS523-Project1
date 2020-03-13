package main

import (
	"flag"
	"fmt"
	"math"
	"sync"
	"time"
)

var circuitID int
var testCircuit *TestCircuit

/*func init(){
	fmt.Println("JE SUIS INIT")
	flag.IntVar(&circuitID,"id",1,"ID between 1 and 8 of the template circuit (default 1)")
	flag.Parse()

	if circuitID <= 0 || circuitID > 8{
		panic("Invalid argument: ID must be between 1 and 8")
	}

	testCircuit = TestCircuits[circuitID-1]
}*/

func main() {
	flag.IntVar(&circuitID, "id", 1, "ID between 1 and 8 of the template circuit (default 1)")
	flag.Parse()

	if circuitID <= 0 || circuitID > 9 {
		panic("Invalid argument: ID must be between 1 and 8")
	}

	testCircuit = TestCircuits[circuitID-1]
	wg := sync.WaitGroup{}
	wg.Add(len(testCircuit.Peers))

	testCircuit.Peers[PartyID(math.MaxUint64)] = ThirdPartyAddr
	lp, err := NewLocalParty(ThirdPartyID, testCircuit.Peers)
	check(err)

	beaverProtocol := lp.NewBeaverProtocol()
	beaverProtocol.BindNetwork(nw)

	for partyID, _ := range testCircuit.Peers {
		go func(id PartyID) {

			defer wg.Done()
			partyInput := testCircuit.Inputs[id][GateID(id)]
			// Create a local party
			lp, err := NewLocalParty(id, testCircuit.Peers)
			check(err)

			// Create the network for the circuit
			network, err := NewTCPNetwork(lp)
			check(err)

			// Connect the circuit network
			err = network.Connect(lp)
			check(err)
			fmt.Println(lp, "connected")
			<-time.After(time.Second) // Leave time for others to connect

			// Create a new circuit evaluation protocol
			protocol := lp.NewProtocol(partyInput, testCircuit.Circuit)
			// Bind evaluation protocol to the network
			protocol.BindNetwork(network)

			// Evaluate the circuit
			protocol.Run()

			fmt.Println(lp, "completed with output", protocol.Output, "where expected is", testCircuit.ExpOutput)

		}(partyID)
	}

	wg.Wait()
}

/*func main() {
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
	peers := map[PartyID]string{
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
	<-time.After(time.Second) // Leave time for others to connect

	// Create a new circuit evaluation protocol
	dummyProtocol := lp.NewDummyProtocol(partyInput)
	// Bind evaluation protocol to the network
	dummyProtocol.BindNetwork(network)

	// Evaluate the circuit
	dummyProtocol.Run()

	fmt.Println(lp, "completed with output", dummyProtocol.Output)
}*/
