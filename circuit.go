package main

type Circuit struct{
	Peers     map[PartyID]string            // Mapping from PartyID to network addresses
	Inputs    map[PartyID]map[GateID]uint64 // The partys' input for each gate
	Circuit   []Operation					// Circuit definition
}



