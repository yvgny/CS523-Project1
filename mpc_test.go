package main

import (
	"fmt"
	"sync"
	"testing"
)

// Iterate through all the circuits defined in test_circuit.go to verify the computation
func TestEval(t *testing.T) {
	for i, testCase := range TestCircuits {
		t.Run(fmt.Sprintf("circuit%d", i+1), func(t *testing.T) {
			N := len(testCase.Peers)
			localParties := make([]*LocalParty, N, N)
			protocol := make([]*Protocol, N, N)
			beaverProtocol := make([]*BeaverProtocol, N, N)

			beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
			for peerID := range testCase.Peers {
				beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
			}

			var err error
			wg := new(sync.WaitGroup)

			for i := range testCase.Peers {
				localParties[i], err = NewLocalParty(i, testCase.Peers)

				if err != nil {
					t.Errorf("creation of new local party failed")
				}

				localParties[i].WaitGroup = wg
				beaverProtocol[i] = localParties[i].NewBeaverProtocol(Params)

			}

			network := GetTestingTCPNetwork(localParties)

			for i, lp := range localParties {
				lp.BindNetwork(network[i])
			}

			wg2 := new(sync.WaitGroup)

			for _, p := range beaverProtocol {
				wg2.Add(1)

				go func(bp *BeaverProtocol, group *sync.WaitGroup, bt map[PartyID]map[WireID]BeaverTriplet) {
					defer group.Done()
					ComputeBeaverTripletHE(bp, bt, testCase.Circuit)
				}(p, wg2, beaverTriplets)
			}
			wg2.Wait()


			for i, lp := range localParties {
				protocol[i] = lp.NewProtocol(testCase.Inputs[lp.ID][GateID(i)], testCase.Circuit, beaverTriplets[lp.ID])
			}

			for _, p := range protocol {
				p.Add(1)
				go func(protocol *Protocol) {
					defer protocol.Done()
					protocol.Run()
				}(p)
			}

			wg.Wait()

			for _, p := range protocol {
				if p.Output != testCase.ExpOutput {
					t.Errorf(p.LocalParty.String(), "result", p.Output, "expected", testCase.ExpOutput)
				}
			}

		})
	}
}
