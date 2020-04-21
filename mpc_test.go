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

func TestTrustedThirdParty(t *testing.T) {
	for i, testCase := range TestCircuits {
		t.Run(fmt.Sprintf("circuit%d", i+1), func(t *testing.T) {
			N := len(testCase.Peers)
			localParties := make([]*LocalParty, N, N)
			protocol := make([]*Protocol, N, N)

			beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
			for peerID := range testCase.Peers {
				beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
			}

			for _, op := range testCase.Circuit {
				if triplets := op.BeaverTriplet(len(testCase.Peers)); triplets != nil {
					for id, triplet := range triplets {
						beaverTriplets[PartyID(id)][op.Output()] = triplet
					}
				}
			}

			var err error
			wg := new(sync.WaitGroup)

			for i := range testCase.Peers {
				localParties[i], err = NewLocalParty(i, testCase.Peers)

				if err != nil {
					t.Errorf("creation of new local party failed")
				}

				localParties[i].WaitGroup = wg

			}

			network := GetTestingTCPNetwork(localParties)

			for i, lp := range localParties {
				lp.BindNetwork(network[i])
			}

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

func BenchmarkPreProcessOneMult3P(b *testing.B) {

	nbrPeers := 20

	benchmarks := make([]TestCircuit, nbrPeers)

	for n := range benchmarks {
		circuit := TestCircuit{}

		peers := make(map[PartyID]string)
		inputs := make(map[PartyID]map[GateID]uint64)
		cir := make([]Operation, 0)
		port := PartyID(1025)
		for i := PartyID(0); i < PartyID(n+1); i++ {
			peers[i] = "localhost:" + string(i+port)
			inputs[i] = map[GateID]uint64{GateID(i): uint64(7)}
			cir = append(cir, &Input{
				Party: i,
				Out:   WireID(i),
			})
		}

		circuit.Peers = peers
		circuit.Inputs = inputs
		circuit.Circuit = append(cir, &Mult{
			In1: 0,
			In2: 1,
			Out: WireID(n + 1),
		}, &Reveal{
			In:  WireID(n + 1),
			Out: WireID(n + 2),
		})
		circuit.ExpOutput = 49

		benchmarks[n] = circuit

	}

	for _, bench := range benchmarks {
		b.Run(fmt.Sprintf("3P: %d peers", len(bench.Peers)), func(b *testing.B) {
			beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
			for peerID := range bench.Peers {
				beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
			}

			for _, op := range bench.Circuit {
				if triplets := op.BeaverTriplet(len(bench.Peers)); triplets != nil {
					for id, triplet := range triplets {
						beaverTriplets[PartyID(id)][op.Output()] = triplet
					}
				}
			}
		})
	}
}

func BenchmarkPreProcessOneMultHE(b *testing.B) {

	nbrPeers := 20

	type Benchmarks struct {
		circuit        TestCircuit
		beaverTriplets map[PartyID]map[WireID]BeaverTriplet
		beaverProtocol []*BeaverProtocol
	}

	bs := make([]Benchmarks, nbrPeers)

	for n := range bs {
		circuit := TestCircuit{}

		peers := make(map[PartyID]string)
		inputs := make(map[PartyID]map[GateID]uint64)
		cir := make([]Operation, 0)
		port := PartyID(1025)
		for i := PartyID(0); i < PartyID(n+1); i++ {
			peers[i] = fmt.Sprintf("localhost:%d", i+port)
			inputs[i] = map[GateID]uint64{GateID(i): uint64(7)}
			cir = append(cir, &Input{
				Party: i,
				Out:   WireID(i),
			})
		}

		circuit.Peers = peers
		circuit.Inputs = inputs
		circuit.Circuit = append(cir, &Mult{
			In1: 0,
			In2: 1,
			Out: WireID(n + 1),
		}, &Reveal{
			In:  WireID(n + 1),
			Out: WireID(n + 2),
		})
		circuit.ExpOutput = 49

		bs[n].circuit = circuit

	}

	for _, bench := range bs {
		N := len(bench.circuit.Peers)
		localParties := make([]*LocalParty, N, N)
		beaverProtocol := make([]*BeaverProtocol, N, N)
		bench.beaverProtocol = beaverProtocol

		beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
		bench.beaverTriplets = beaverTriplets
		for peerID := range bench.circuit.Peers {
			beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
		}

		var err error
		wg := new(sync.WaitGroup)

		for i := range bench.circuit.Peers {
			localParties[i], err = NewLocalParty(i, bench.circuit.Peers)

			if err != nil {
				b.Errorf("creation of new local party failed")
			}

			localParties[i].WaitGroup = wg
			beaverProtocol[i] = localParties[i].NewBeaverProtocol(Params)

		}

		network := GetTestingTCPNetwork(localParties)

		for i, lp := range localParties {
			lp.BindNetwork(network[i])
		}

		b.Run(fmt.Sprintf("HE: %d peers", len(bench.circuit.Peers)), func(b *testing.B) {
			wg2 := new(sync.WaitGroup)
			b.ResetTimer()
			for _, p := range bench.beaverProtocol {
				wg2.Add(1)

				go func(bp *BeaverProtocol, group *sync.WaitGroup, bt map[PartyID]map[WireID]BeaverTriplet) {
					defer group.Done()
					ComputeBeaverTripletHE(bp, bt, bench.circuit.Circuit)
				}(p, wg2, bench.beaverTriplets)
			}
			wg2.Wait()
		})
	}
}

func BenchmarkOperations(b *testing.B) {

	testCases := make([]TestCircuit, 5)
	circuits := map[int][]Operation{
		0: {
			&Input{
				Party: 0,
				Out:   0,
			},
			&Input{
				Party: 1,
				Out:   1,
			},
			&Add{
				In1: 0,
				In2: 1,
				Out: 2,
			},
			&Reveal{
				In:  2,
				Out: 3,
			},
		},
		1: {
			&Input{
				Party: 0,
				Out:   0,
			},
			&Input{
				Party: 1,
				Out:   1,
			},
			&AddCst{
				In:       0,
				CstValue: 1,
				Out:      2,
			},
			&Reveal{
				In:  2,
				Out: 3,
			},
		},
		2: {
			&Input{
				Party: 0,
				Out:   0,
			},
			&Input{
				Party: 1,
				Out:   1,
			},
			&Sub{
				In1: 0,
				In2: 1,
				Out: 2,
			},
			&Reveal{
				In:  2,
				Out: 3,
			},
		},
		3: {
			&Input{
				Party: 0,
				Out:   0,
			},
			&Input{
				Party: 1,
				Out:   1,
			},
			&Mult{
				In1: 0,
				In2: 1,
				Out: 2,
			},
			&Reveal{
				In:  2,
				Out: 3,
			},
		},
		4: {
			&Input{
				Party: 0,
				Out:   0,
			},
			&Input{
				Party: 1,
				Out:   1,
			},
			&MultCst{
				In:       0,
				CstValue: 1,
				Out:      2,
			},
			&Reveal{
				In:  2,
				Out: 3,
			},
		},
	}

	names := map[int]string{
		0: "Add",
		1: "AddCst",
		2: "Sub",
		3: "Mult",
		4: "MultCst",
	}

	for i := range testCases {
		testCases[i] = TestCircuit{
			Peers: map[PartyID]string{
				0: "localhost:6660",
				1: "localhost:6661",
			},
			Inputs: map[PartyID]map[GateID]uint64{
				0: {0: 11},
				1: {1: 8},
			},
			Circuit:   circuits[i],
			ExpOutput: 0,
		}
	}

	for i, testCase := range testCases {
		b.Run(names[i], func(b *testing.B) {
			N := len(testCase.Peers)
			localParties := make([]*LocalParty, N, N)
			protocol := make([]*Protocol, N, N)

			beaverTriplets := make(map[PartyID]map[WireID]BeaverTriplet)
			for peerID := range testCase.Peers {
				beaverTriplets[peerID] = make(map[WireID]BeaverTriplet)
			}

			for _, op := range testCase.Circuit {
				if triplets := op.BeaverTriplet(len(testCase.Peers)); triplets != nil {
					for id, triplet := range triplets {
						beaverTriplets[PartyID(id)][op.Output()] = triplet
					}
				}
			}

			var err error
			wg := new(sync.WaitGroup)

			for i := range testCase.Peers {
				localParties[i], err = NewLocalParty(i, testCase.Peers)

				if err != nil {
					b.Errorf("creation of new local party failed")
				}

				localParties[i].WaitGroup = wg

			}

			network := GetTestingTCPNetwork(localParties)

			for i, lp := range localParties {
				lp.BindNetwork(network[i])
			}

			for i, lp := range localParties {
				protocol[i] = lp.NewProtocol(testCase.Inputs[lp.ID][GateID(i)], testCase.Circuit, beaverTriplets[lp.ID])
			}
			b.ResetTimer()
			for _, p := range protocol {
				p.Add(1)
				go func(protocol *Protocol) {
					defer protocol.Done()
					protocol.Run()
				}(p)
			}

			wg.Wait()

		})
	}
}
