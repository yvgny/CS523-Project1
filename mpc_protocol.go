package main

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
)

var q = big.NewInt(1<<16 + 1)

type MPCMessage struct {
	Out   WireID
	Value uint64
}

type Protocol struct {
	*LocalParty
	Peers map[PartyID]*Remote

	Input          uint64
	Output         uint64
	Circuit        Circuit
	WireOutput     map[WireID]*big.Int
	BeaverTriplets map[WireID]BeaverTriplet
}

type Remote struct {
	*RemoteParty
	Chan        chan MPCMessage
	ReceiveChan chan MPCMessage
}

func (lp *LocalParty) NewProtocol(input uint64, circuit Circuit, beaverTriplets map[WireID]BeaverTriplet) *Protocol {
	cep := new(Protocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*Remote, len(lp.Peers))
	cep.WireOutput = make(map[WireID]*big.Int)
	cep.BeaverTriplets = beaverTriplets
	cep.Circuit = circuit
	for i, rp := range lp.Peers {
		cep.Peers[i] = &Remote{
			RemoteParty: rp,
			Chan:        make(chan MPCMessage, 32),
			ReceiveChan: make(chan MPCMessage, 32),
		}
	}

	cep.Input = input
	return cep
}

func (cep *Protocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID {
			continue
		}

		rp := cep.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *Remote) {
			for {
				var val uint64
				var out WireID
				var err error
				err = binary.Read(conn, binary.BigEndian, &val)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &out)
				check(err)
				msg := MPCMessage{
					Value: val,
					Out:   out,
				}
				//fmt.Println(cep, "receiving", msg, "from", rp)
				rp.ReceiveChan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *Remote) {
			var m MPCMessage
			var open = true
			for open {
				m, open = <-rp.Chan
				//fmt.Println(cep, "sending", m, "to", rp)
				check(binary.Write(conn, binary.BigEndian, m.Value))
				check(binary.Write(conn, binary.BigEndian, m.Out))
			}
		}(conn, rp)
	}
}

func (cep *Protocol) Run() {

	fmt.Println(cep, "is running")

	for _, op := range cep.Circuit {
		op.Eval(cep)
	}

	cep.Output = cep.WireOutput[cep.Circuit[len(cep.Circuit)-1].Output()].Uint64()

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

}
