package param

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/EducationEKT/EKT/core/types"
)

var LocalNet = []types.Peer{
	{"968b10ebc111ea3434de7333d82e54890c4a2d8c34577e0e54f3464eb88e3b2f", "127.0.0.1", 19951, 4},
}

func init() {
	loadLocalNet()
}

func loadLocalNet() {
	cfg := "localnet.json"
	data, err := ioutil.ReadFile(cfg)
	if err != nil {
		return
	}
	log.Println("Found localnet.json, loading it")
	peers := [][]interface{}{}
	err = json.Unmarshal(data, &peers)
	if err != nil {
		log.Println("Invalid localnet.json format, ingore it")
		return
	}
	net := []types.Peer{}
	for _, peer := range peers {
		if len(peer) != 4 {
			fmt.Println("Invalid localnet.json format, ingore it")
			return
		}
		net = append(net, types.Peer{peer[0].(string), peer[1].(string), int32(peer[2].(float64)), int(peer[3].(float64))})
	}
	LocalNet = net
	log.Println("Using localnet.json")
}
