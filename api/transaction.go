package api

import (
	"encoding/json"

	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/encapdb"
	"github.com/EducationEKT/EKT/node"
	"github.com/EducationEKT/xserver/x_err"
	"github.com/EducationEKT/xserver/x_http/x_req"
	"github.com/EducationEKT/xserver/x_http/x_resp"
	"github.com/EducationEKT/xserver/x_http/x_router"
)

func init() {
	x_router.Get("/transaction/api/fee", fee)
	x_router.Post("/transaction/api/newTransaction", broadcast, newTransaction)
	x_router.Get("/transaction/api/userTxs", userTxs)
	x_router.Get("/transaction/api/getReceiptByTxHash", getReceiptByTxHash)
}

func fee(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	return x_resp.Return(node.SuggestFee(), nil)
}

func userTxs(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	address := req.MustGetString("address")
	return x_resp.Return(node.GetMainChain().Pool.GetUserTxs(address), nil)
}

func newTransaction(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	var tx userevent.Transaction
	err := json.Unmarshal(req.Body, &tx)
	if err != nil {
		return nil, x_err.New(-1, err.Error())
	}
	if !userevent.ValidateTransaction(tx) {
		return nil, x_err.New(-401, "error signature")
	}
	if node.GetMainChain().NewTransaction(&tx) {
		db.GetDBInst().Set(tx.TxId(), tx.Bytes())
	}
	return x_resp.Return(tx.TransactionId(), err)
}

func getReceiptByTxHash(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	hash := req.MustGetString("hash")
	return x_resp.Return(encapdb.GetReceiptByTxHash(node.GetMainChain().ChainId, hash), nil)
}
