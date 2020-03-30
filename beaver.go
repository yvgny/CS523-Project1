package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/ldsec/lattigo/bfv"
)

type BeaverMessage struct {
	Value []uint64
}

type BeaverProtocol struct {
	*LocalParty
	Params *bfv.Parameters
	Peers  map[PartyID]*BeaverRemoteParty
}

type BeaverRemoteParty struct {
	*RemoteParty
	Chan        chan BeaverMessage
	ReceiveChan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol(params *bfv.Parameters) *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Peers = make(map[PartyID]*BeaverRemoteParty, len(lp.Peers))
	cep.Params = params
	for i, rp := range lp.Peers {
		cep.Peers[i] = &BeaverRemoteParty{
			RemoteParty: rp,
			Chan:        make(chan BeaverMessage, 32),
			ReceiveChan: make(chan BeaverMessage, 32),
		}
	}

	return cep
}

func (cep *BeaverProtocol) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == cep.ID {
			continue
		}

		rp := cep.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *BeaverRemoteParty) {
			for {
				var val uint64
				var out WireID
				var err error
				err = binary.Read(conn, binary.BigEndian, &val)
				check(err)
				err = binary.Read(conn, binary.BigEndian, &out)
				check(err)
				msg := BeaverMessage{
					Value: val,
					Out:   out,
				}
				//fmt.Println(cep, "receiving", msg, "from", rp)
				rp.ReceiveChan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *BeaverRemoteParty) {
			var m BeaverMessage
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

func (cep *BeaverProtocol) Run() {

	fmt.Println(cep, "is running")

	// TODO

	if cep.WaitGroup != nil {
		cep.WaitGroup.Done()
	}

}
