package main

import (
	"fmt"
	"github.com/ldsec/lattigo/bfv"
	"github.com/ldsec/lattigo/ring"
	"sync"
	"testing"
)

var params = bfv.DefaultParams[bfv.PN13QP218]

func TestEval(t *testing.T) {
	for i , testCase := range TestCircuits{
		t.Run(fmt.Sprintf("circuit%d", i+1), func(t *testing.T) {
			N := len(testCase.Peers)
			localParties := make([]*LocalParty, N, N)
			protocol := make([]*Protocol, N, N)
			beaverProtocol := make([]*BeaverProtocol, N, N)

			beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
			for peerID, _ := range testCase.Peers {
				beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
			}



			var err error
			wg := new(sync.WaitGroup)

			for i:= range testCase.Peers{
				localParties[i], err = NewLocalParty(i, testCase.Peers)

				if err != nil{
					t.Errorf("creation of new local party failed")
				}

				localParties[i].WaitGroup = wg
				beaverProtocol[i] = localParties[i].NewBeaverProtocol(params)

			}

			network := GetTestingTCPNetwork(localParties)

			for i, lp := range localParties{
				lp.BindNetwork(network[i])
			}

			for _, bp := range beaverProtocol{

				var currIndex uint64 = 0
				var triplet Triplets
				for _, op := range testCase.Circuit {
					if op.BeaverTriplet(len(testCase.Peers)) {
						if currIndex%(1<<params.LogN) == 0 {
							bp.Run()
							triplet = bp.BeaverTriplets
							currIndex = 0
						}
						beaverTriplets[bp.ID][op.Output()] = BeaverTriplet{
							a: ring.NewUint(triplet.ai[currIndex]),
							b: ring.NewUint(triplet.bi[currIndex]),
							c: ring.NewUint(triplet.ci[currIndex]),
						}
					}
				}
			}

			fmt.Println("yooo")

			for i, lp:= range localParties{
				protocol[i] = lp.NewProtocol(testCase.Inputs[lp.ID][GateID(i)], testCase.Circuit, beaverTriplets[lp.ID])
			}

			for _, p := range protocol{
				p.Add(1)
				go p.Run()
			}

			wg.Wait()

			for _, p := range protocol {
				if p.Output != testCase.ExpOutput {
					t.Errorf(p.LocalParty.String(), "result", p.Output, "expected", testCase.ExpOutput)
				}
			}

		})
	}
	/*for i, testCircuit := range TestCircuits {

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
			if p.Output != testCircuit.ExpOutput{
				t.FailNow()
			}
		}

		fmt.Printf("Test for Circuit %d passed\n", i+1)

	}

	fmt.Println("test completed")*/
}
