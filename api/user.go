package api

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/node"
	"github.com/EducationEKT/xserver/x_err"
	"github.com/EducationEKT/xserver/x_http/x_req"
	"github.com/EducationEKT/xserver/x_http/x_resp"
	"github.com/EducationEKT/xserver/x_http/x_router"
)

func init() {
	x_router.Get("/account/api/info", userInfo)
	x_router.Get("/account/api/nonce", userNonce)
}

func userInfo(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	address := req.MustGetString("address")
	hexAddress, err := hex.DecodeString(address)
	if err != nil {
		return x_resp.Return(nil, err)
	}
	account, err := node.GetMainChain().LastHeader().GetAccount(hexAddress)
	if err != nil {
		return x_resp.Return(nil, err)
	}
	return x_resp.Return(account, nil)
}

func userNonce(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	hexAddress := req.MustGetString("address")

	txs := node.GetMainChain().Pool.GetUserTxs(hexAddress)
	if txs != nil {
		return x_resp.Return(txs.Nonce, nil)
	}

	address, err := hex.DecodeString(hexAddress)
	if err != nil {
		return x_resp.Return(nil, err)
	}
	// get user nonce by user stat tree
	account, err := node.GetMainChain().LastHeader().GetAccount(address)
	if err != nil {
		return x_resp.Return(nil, err)
	}
	nonce := account.GetNonce()

	return x_resp.Return(nonce, nil)
}
