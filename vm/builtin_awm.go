package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/EducationEKT/EKT/core/userevent"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

func builtinAWMVM_Sha3_256(call FunctionCall) Value {
	param := call.Argument(0).string()
	return toValue_string(hex.EncodeToString(crypto.Sha3_256([]byte(param))))
}

func builtinAWMVM_ecrecover(call FunctionCall) Value {
	msg := call.Argument(0).string()
	sign := call.Argument(1).string()
	msg_b, err := hex.DecodeString(msg)
	if err != nil {
		return toValue_string("")
	}
	sign_b, err := hex.DecodeString(sign)
	if err != nil {
		return toValue_string("")
	}
	pubKey, err := crypto.RecoverPubKey(msg_b, sign_b)
	if err != nil {
		return toValue_string("")
	}
	return toValue_string(hex.EncodeToString(types.FromPubKeyToAddress(pubKey)))
}

func builtinAWMVM_verify(call FunctionCall) Value {
	msg := call.Argument(0).string()
	sign := call.Argument(1).string()
	address := call.Argument(2).string()
	msg_b, err := hex.DecodeString(msg)
	if err != nil {
		return toValue_bool(false)
	}
	sign_b, err := hex.DecodeString(sign)
	if err != nil {
		return toValue_bool(false)
	}
	pubKey, err := crypto.RecoverPubKey(msg_b, sign_b)
	if err != nil {
		return toValue_bool(false)
	}
	address_b, err := hex.DecodeString(address)
	if err != nil {
		return toValue_bool(false)
	}
	return toValue_bool(bytes.EqualFold(types.FromPubKeyToAddress(pubKey), address_b))
}

func builtinAWMVM_Contract_Refuse_Tx(call FunctionCall) Value {
	data := call.Argument(0).string()
	var tx userevent.Transaction
	err := json.Unmarshal([]byte(data), &tx)
	if err != nil {
		return toValue_string("")
	}
	subTx := userevent.NewSubTransaction(tx.TxId(), tx.To, tx.From, tx.Amount, "contract refused", tx.TokenAddress)
	txData, _ := json.Marshal(subTx)
	return toValue_string(string(txData))
}
