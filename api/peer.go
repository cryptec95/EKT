package api

import (
	"fmt"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/param"
	"github.com/EducationEKT/EKT/util"

	"github.com/EducationEKT/xserver/x_err"
	"github.com/EducationEKT/xserver/x_http/x_req"
	"github.com/EducationEKT/xserver/x_http/x_resp"
	"github.com/EducationEKT/xserver/x_http/x_router"
)

func init() {
	x_router.All("/peer/api/ping", ping)
	x_router.Post("/peer/api/peers", delegatePeers)
}

func delegatePeers(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	peers := param.MainChainDelegateNode
	return x_resp.Success(peers), x_err.NewXErr(nil)
}

func ping(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	resp := &x_resp.XRespContainer{
		HttpCode: 200,
		Body:     []byte("pong"),
	}
	return resp, nil
}

func broadcast(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	if len(req.Query) == 0 {
		for _, peer := range param.MainChainDelegateNode {
			if !peer.Equal(conf.EKTConfig.Node) {
				url := fmt.Sprintf(`http://%s:%d%s?broadcast=true`, peer.Address, peer.Port, req.Path)
				util.HttpPost(url, req.Body)
			}
		}
	}
	return nil, nil
}
