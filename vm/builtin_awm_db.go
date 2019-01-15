package vm

import "encoding/hex"

const AWM_DB_PREFIX = "_AWM_CONTRACT_DB_"

func builtin_awm_db_get(call FunctionCall) Value {
	if call.Otto.tx == nil {
		return falseValue
	}
	contractAddr := hex.EncodeToString(call.Otto.tx.To)
	key := AWM_DB_PREFIX + contractAddr + call.Argument(0).string()
	value, err := call.Otto.db.Get([]byte(key))
	if err != nil {
		return toValue_string("")
	}
	return toValue_string(string(value))
}

func builtin_awm_db_set(call FunctionCall) Value {
	if call.Otto.tx == nil {
		return falseValue
	}
	contractAddr := hex.EncodeToString(call.Otto.tx.To)
	key := AWM_DB_PREFIX + contractAddr + call.Argument(0).string()
	value := call.Argument(1).string()
	err := call.Otto.db.Set([]byte(key), []byte(value))
	if err != nil {
		return toValue_bool(false)
	}
	return toValue_bool(true)
}

func builtin_awm_db_delete(call FunctionCall) Value {
	if call.Otto.tx == nil {
		return falseValue
	}
	contractAddr := hex.EncodeToString(call.Otto.tx.To)
	key := AWM_DB_PREFIX + contractAddr + call.Argument(0).string()
	err := call.Otto.db.Delete([]byte(key))
	if err != nil {
		return toValue_bool(false)
	}
	return toValue_bool(true)
}
