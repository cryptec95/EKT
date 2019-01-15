package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/log"
)

func builtinAWM_Sha3_256(call FunctionCall) Value {
	param := call.Argument(0).string()
	return toValue_string(hex.EncodeToString(crypto.Sha3_256([]byte(param))))
}

func builtinAWM_secp256k1_ecrecover(call FunctionCall) Value {
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

func builtinAWM_secp256k1_verify(call FunctionCall) Value {
	msg := call.Argument(0).string()
	sign := call.Argument(1).string()
	address := call.Argument(2).string()
	msg_b := crypto.Sha3_256([]byte(msg))
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
	return toValue_bool(bytes.Equal(types.FromPubKeyToAddress(pubKey), address_b))
}

func builtinAWM_Contract_Refuse_Tx(call FunctionCall) Value {
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

func builtinAWM_contract_call(call FunctionCall) Value {
	if len(call.ArgumentList) < 2 {
		return falseValue
	}

	vm := call.Otto.clone()
	contractAddr := call.ArgumentList[0].String()
	if strings.HasPrefix(contractAddr, "0x") {
		contractAddr = contractAddr[2:]
	}

	address, err := hex.DecodeString(contractAddr)
	if err != nil {
		return falseValue
	}

	err = vm.LoadContract(address)
	if err != nil {
		return falseValue
	}

	method := call.ArgumentList[1]
	args := call.ArgumentList[2:]

	log.LogErr(vm.Set("args", args))
	log.LogErr(vm.Set("sender", hex.EncodeToString(call.Otto.tx.To)))

	_, err = vm.Run(fmt.Sprintf("var result = %s.apply(null, args);", method.String()))
	if err != nil {
		return falseValue
	}

	value, err := vm.Get("result")
	if err != nil {
		return falseValue
	}

	data := vm.contractData()
	log.LogErr(vm.chain.ModifyContract(address, data))

	return value
}
