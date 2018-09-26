package types

import "sync"

type Round struct {
	CurrentIndex int          `json:"currentIndex"` // default -1
	Peers        []Peer       `json:"peers"`
	time         int64        `json:"-"`
	Locker       sync.RWMutex `json:"-"`
}

func NewRound(peers Peers, index int, time int64) *Round {
	return &Round{
		CurrentIndex: index,
		Peers:        peers,
		time:         time,
		Locker:       sync.RWMutex{},
	}
}

func (round *Round) SetTime(time int64) {
	round.Locker.Lock()
	round.time = time
	round.Locker.Unlock()
}

func (round Round) GetTime() int64 {
	return round.time
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

func (round Round) Clone() Round {
	round.Locker.RLock()
	defer round.Locker.RUnlock()
	return Round{
		CurrentIndex: round.CurrentIndex,
		Peers:        round.Peers,
		time:         round.time,
	}
}
