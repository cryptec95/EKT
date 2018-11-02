package vm

import (
	"bytes"
	"encoding/hex"

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
