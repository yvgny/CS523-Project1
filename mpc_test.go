package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestMPCProtocol(t *testing.T) {
	for _, testCircuit := range TestCircuits {

		N := uint64(len(testCircuit.Peers))
		P := make([]*LocalParty, N, N)
		protocol := make([]*Protocol, N, N)

		var err error
		wg := new(sync.WaitGroup)
		for i := range testCircuit.Peers {
			P[i], err = NewLocalParty(i, testCircuit.Peers)
			P[i].WaitGroup = wg
			check(err)

			protocol[i] = P[i].NewProtocol(testCircuit.Inputs[i][GateID(i)], testCircuit.Circuit)
		}

		network := GetTestingTCPNetwork(P)
		fmt.Println("parties connected")

		for i, Pi := range protocol {
			Pi.BindNetwork(network[i])
		}

		for _, p := range protocol {
			p.Add(1)
			go p.Run()
		}
		wg.Wait()

		for _, p := range protocol {
			fmt.Println(p, "completed with output", p.Output)
		}

	}

	fmt.Println("test completed")
}
