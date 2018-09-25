package types

type Round struct {
	CurrentIndex int    `json:"currentIndex"` // default -1
	Peers        []Peer `json:"peers"`
}

func (round *Round) UpdateIndex(miner string) {
	round.CurrentIndex = round.IndexOf(miner)
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
