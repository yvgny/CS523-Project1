package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/ldsec/lattigo/ring"
)

func main() {
	var circuitID int
	var testCircuit *TestCircuit
	var centralized bool

	flag.IntVar(&circuitID, "id", 1, fmt.Sprintf("ID between 1 and %d of the template circuit", len(TestCircuits)))
	flag.BoolVar(&centralized, "c", false, "Use a centralized generation of beaver triplets")

	flag.Parse()

	if circuitID <= 0 || circuitID > len(TestCircuits) {
		panic(fmt.Sprintf("Invalid argument: ID must be between 1 and %d", len(TestCircuits)))
	}

	testCircuit = TestCircuits[circuitID-1]

	beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)

	for peerID := range testCircuit.Peers {
		beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
	}

	if centralized {
		for _, op := range testCircuit.Circuit {
			if triplets := op.BeaverTriplet(len(testCircuit.Peers)); triplets != nil {
				for id, triplet := range triplets {
					beaverTriplets[PartyID(id)][op.Output()] = triplet
				}
			}
		}
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(testCircuit.Peers))

	for partyID := range testCircuit.Peers {
		go func(id PartyID) {

			//defer wg.Done()
			partyInput := testCircuit.Inputs[id][GateID(id)]
			// Create a local party
			lp, err := NewLocalParty(id, testCircuit.Peers)
			check(err)
			lp.WaitGroup = wg

			// Create the network for the circuit
			network, err := NewTCPNetwork(lp)
			check(err)

			// Connect the circuit network
			err = network.Connect(lp)
			check(err)
			fmt.Println(lp, "connected")
			<-time.After(time.Second) // Leave time for others to connect

			lp.BindNetwork(network)

			if !centralized {
				beaverProtocol := lp.NewBeaverProtocol(Params)
				ComputeBeaverTripletHE(beaverProtocol, beaverTriplets, testCircuit.Circuit)
			}

			// Create a new circuit evaluation protocol
			protocol := lp.NewProtocol(partyInput, testCircuit.Circuit, beaverTriplets[id])

			// Evaluate the circuit
			protocol.Run()
		}(partyID)
	}
	wg.Wait()
}

func ComputeBeaverTripletHE(beaverProtocol *BeaverProtocol, beaverTriplets map[PartyID]map[WireID]BeaverTriplet, circuit Circuit) {

	var currIndex uint64 = 0
	var triplet Triplets
	for _, op := range circuit {
		if op.IsMult() {
			if currIndex%(1<<Params.LogN) == 0 {
				beaverProtocol.Run()
				triplet = beaverProtocol.BeaverTriplets
				currIndex = 0
			}
			beaverTriplets[beaverProtocol.ID][op.Output()] = BeaverTriplet{
				a: ring.NewUint(triplet.ai[currIndex]),
				b: ring.NewUint(triplet.bi[currIndex]),
				c: ring.NewUint(triplet.ci[currIndex]),
			}
		}
	}
}
