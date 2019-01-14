package types

type Round struct {
	Peers []Peer `json:"peers"`
}

func NewRound(peers Peers) *Round {
	return &Round{
		Peers: peers,
	}
}

func (round Round) Distance(previous, my string) int {
	index1 := round.IndexOf(previous)
	index2 := round.IndexOf(my)
	distance := (index2 - index1 + round.Len()) % round.Len()
	return distance
}

func (round Round) IndexOf(miner string) int {
	for i := 0; i < round.Len(); i++ {
		if round.Peers[i].Account == miner {
			return i
		}
	}
	return -1
}

func (round Round) Len() int {
	return len(round.Peers)
}

func (round Round) Clone() Round {
	return Round{
		Peers: round.Peers,
	}
}
