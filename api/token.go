package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/EducationEKT/EKT/blockchain_manager"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/param"
	"github.com/EducationEKT/EKT/util"
	"github.com/EducationEKT/xserver/x_err"
	"github.com/EducationEKT/xserver/x_http/x_req"
	"github.com/EducationEKT/xserver/x_http/x_resp"
	"github.com/EducationEKT/xserver/x_http/x_router"
)

func init() {
	x_router.All("/token/api/issue", tokenIssue)
}

func tokenIssue(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	return x_resp.Fail(-1, "not open", nil), nil
}

func issueToken(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	var tokenIssue userevent.TokenIssue
	err := json.Unmarshal(req.Body, &tokenIssue)
	if err != nil {
		return x_resp.Fail(-1, "error request", nil), nil
	}
	if userevent.Validate(&tokenIssue) {
		db.GetDBInst().Set(crypto.Sha3_256(tokenIssue.Bytes()), tokenIssue.Bytes())
		blockchain_manager.GetMainChain().NewUserEvent(&tokenIssue)
		return x_resp.Success(hex.EncodeToString(tokenIssue.Token.Address())), nil
	}
	return x_resp.Fail(-1, "error signature", nil), nil
}

func broadcastTokenIssue(req *x_req.XReq) (*x_resp.XRespContainer, *x_err.XErr) {
	if len(req.Query) == 0 {
		for _, peer := range param.MainChainDelegateNode {
			if !peer.Equal(conf.EKTConfig.Node) {
				url := fmt.Sprintf(`http://%s:%d/token/api/issue?broadcast=true`, peer.Address, peer.Port)
				util.HttpPost(url, req.Body)
			}
		}
	}
	return nil, nil
}
