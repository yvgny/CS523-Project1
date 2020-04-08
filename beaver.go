package main

import (
	"errors"
	"fmt"

	"github.com/ldsec/lattigo/bfv"
	"github.com/ldsec/lattigo/ring"
)

type BeaverMessage struct {
	Size  uint64
	Value []byte
}

type BeaverProtocol struct {
	*LocalParty
	Params         *bfv.Parameters
	Encoder        bfv.Encoder
	Evaluator      bfv.Evaluator
	BeaverTriplets Triplets
}

type Triplets struct {
	ai   []uint64
	bi   []uint64
	biPt *bfv.Plaintext
	ci   []uint64
	sk   *bfv.SecretKey
}

type BeaverRemoteParty struct {
	*RemoteParty
	Chan        chan BeaverMessage
	ReceiveChan chan BeaverMessage
}

func (lp *LocalParty) NewBeaverProtocol(params *bfv.Parameters) *BeaverProtocol {
	cep := new(BeaverProtocol)
	cep.LocalParty = lp
	cep.Params = params
	cep.Encoder = bfv.NewEncoder(params)
	cep.Evaluator = bfv.NewEvaluator(params)

	return cep
}

func (cep *BeaverProtocol) Run() {

	fmt.Println(cep, "is running")

	cep.GenerateTriplets()
	cep.ReceiveOtherBeaver()
	cep.ComputeC()

}

func (cep *BeaverProtocol) ComputeC() {
	encC := bfv.NewCiphertext(cep.Params, 1)
	for id, peer := range cep.Peers {
		if id != cep.ID {
			msg := <-peer.ReceiveChan
			if msg.BeaverMessage == nil {
				check(errors.New("MPCMessage received instead of BeaverMessage"))
			}
			dij := bfv.NewCiphertext(cep.Params, 1)
			err := dij.UnmarshalBinary(msg.BeaverMessage.Value)
			check(err)
			cep.Evaluator.Add(encC, dij, encC)
		}
	}

	decryptor := bfv.NewDecryptor(cep.Params, cep.BeaverTriplets.sk)
	decC := decryptor.DecryptNew(encC)
	decCVec := cep.Encoder.DecodeUint(decC)
	cep.BeaverTriplets.ci = addVec(cep.BeaverTriplets.ci, decCVec, cep.Params.T)
}

func (cep *BeaverProtocol) ReceiveOtherBeaver() {
	bi := cep.BeaverTriplets.biPt
	for id, peer := range cep.Peers {
		if id != cep.ID {
			msg := <-peer.ReceiveChan
			if msg.BeaverMessage == nil {
				check(errors.New("MPCMessage received instead of BeaverMessage"))
			}
			dj := bfv.NewCiphertext(cep.Params, 1)
			err := dj.UnmarshalBinary(msg.BeaverMessage.Value)
			check(err)

			rij := newRandomVec(1<<cep.Params.LogN, cep.Params.T)

			cep.BeaverTriplets.ci = subVec(cep.BeaverTriplets.ci, rij, cep.Params.T)

			rijPt := bfv.NewPlaintext(cep.Params)
			cep.Encoder.EncodeUint(rij, rijPt)

			mul := bfv.NewCiphertext(cep.Params, 1)
			cep.Evaluator.Mul(dj, bi, mul)

			dij_clean := bfv.NewCiphertext(cep.Params, 1)
			cep.Evaluator.Add(mul, rijPt, dij_clean)

			//TODO: verify errors (ligne 19 et 20 protocol)

			contextQP, err := ring.NewContextWithParams(1<<cep.Params.LogN, cep.Params.Qi)

			// Get value of the ciphertext
			dij_clean_values := dij_clean.Value()
			bound := uint64(cep.Params.Sigma * 6)

			for i, _ := range dij_clean_values {
				// Generate error
				err_poly := contextQP.SampleGaussianNTTNew(cep.Params.Sigma, bound)

				// Add to current polynomial
				res := contextQP.NewPoly()
				contextQP.Add(dij_clean_values[i], err_poly, res)
				dij_clean_values[i] = res
			}

			// Transform back to ciphertext
			dij := bfv.NewCiphertext(cep.Params, 1)
			dij.SetValue(dij_clean_values)

			//
			// End of errors handling
			//

			bytes, err := dij.MarshalBinary()
			check(err)
			beaverMessage := BeaverMessage{Size: uint64(len(bytes)), Value: bytes}
			msg = Message{
				BeaverMessage: &beaverMessage,
			}
			peer.Chan <- msg

		}
	}
}

func (cep *BeaverProtocol) GenerateTriplets() {
	keyGen := bfv.NewKeyGenerator(cep.Params)

	ai := newRandomVec(1<<cep.Params.LogN, cep.Params.T)
	bi := newRandomVec(1<<cep.Params.LogN, cep.Params.T)
	ci := mulVec(ai, bi, cep.Params.T)

	aiPt := bfv.NewPlaintext(cep.Params)
	cep.Encoder.EncodeUint(ai, aiPt)

	biPt := bfv.NewPlaintext(cep.Params)
	cep.Encoder.EncodeUint(bi, biPt)

	ski := keyGen.GenSecretKey()

	cep.BeaverTriplets = Triplets{ai: ai, bi: bi, biPt: biPt, ci: ci, sk: ski}

	encryptor := bfv.NewEncryptorFromSk(cep.Params, ski)
	di := encryptor.EncryptNew(aiPt)
	bytes, err := di.MarshalBinary()
	check(err)
	beaverMessage := BeaverMessage{Size: uint64(len(bytes)), Value: bytes}

	for id, peer := range cep.Peers {
		if id != cep.ID {
			msg := Message{
				BeaverMessage: &beaverMessage,
			}
			peer.Chan <- msg
		}
	}

}
