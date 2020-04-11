package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
)

type PartyID uint64

type Party struct {
	ID   PartyID
	Addr string
}

type LocalParty struct {
	Party
	*sync.WaitGroup
	Peers map[PartyID]*RemoteParty
}

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func NewLocalParty(id PartyID, peers map[PartyID]string) (*LocalParty, error) {
	// Create a new local party from the peers map
	p := &LocalParty{}
	p.ID = id

	p.Peers = make(map[PartyID]*RemoteParty, len(peers))
	p.Addr = peers[id]

	var err error
	for pId, pAddr := range peers {
		p.Peers[pId], err = NewRemoteParty(pId, pAddr)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (lp *LocalParty) String() string {
	// Print the party number
	return fmt.Sprintf("party-%d", lp.ID)
}

type Message struct {
	MPCMessage    *MPCMessage
	BeaverMessage *BeaverMessage
}

type MessageType uint64

const (
	MPC MessageType = iota
	Beaver
)

type RemoteParty struct {
	Party
	Chan        chan Message
	ReceiveChan chan Message
}

func (rp *RemoteParty) String() string {
	return fmt.Sprintf("party-%d", rp.ID)
}

func NewRemoteParty(id PartyID, addr string) (*RemoteParty, error) {
	p := &RemoteParty{}
	p.ID = id
	p.Addr = addr
	p.Chan = make(chan Message, 32)
	p.ReceiveChan = make(chan Message, 32)
	return p, nil
}

func (lp *LocalParty) BindNetwork(nw *TCPNetworkStruct) {
	for partyID, conn := range nw.Conns {

		if partyID == lp.ID {
			continue
		}

		rp := lp.Peers[partyID]

		// Receiving loop from remote
		go func(conn net.Conn, rp *RemoteParty) {
			for {
				var msgType MessageType
				var msg Message
				err := binary.Read(conn, binary.BigEndian, &msgType)
				check(err)
				switch msgType {
				case MPC:
					var val uint64
					var out WireID
					var err error
					err = binary.Read(conn, binary.BigEndian, &val)
					check(err)
					err = binary.Read(conn, binary.BigEndian, &out)
					check(err)
					msg.MPCMessage = &MPCMessage{
						Value: val,
						Out:   out,
					}
				case Beaver:
					var size uint64

					err = binary.Read(conn, binary.BigEndian, &size)
					check(err)
					val := make([]byte, size)
					err = binary.Read(conn, binary.BigEndian, &val)
					check(err)

					msg.BeaverMessage = &BeaverMessage{Size: size, Value: val}
				default:
					check(errors.New("unknown message type"))
				}
				rp.ReceiveChan <- msg
			}
		}(conn, rp)

		// Sending loop of remote
		go func(conn net.Conn, rp *RemoteParty) {
			var m Message
			var open = true
			for open {
				m, open = <-rp.Chan
				if beaverMsg := m.BeaverMessage; beaverMsg != nil {
					check(binary.Write(conn, binary.BigEndian, Beaver))
					check(binary.Write(conn, binary.BigEndian, beaverMsg.Size))
					check(binary.Write(conn, binary.BigEndian, beaverMsg.Value))

				} else if mpcMsg := m.MPCMessage; mpcMsg != nil {
					check(binary.Write(conn, binary.BigEndian, MPC))
					check(binary.Write(conn, binary.BigEndian, mpcMsg.Value))
					check(binary.Write(conn, binary.BigEndian, mpcMsg.Out))
				} else {
					check(errors.New("no message to send"))
				}
			}
		}(conn, rp)
	}
}
