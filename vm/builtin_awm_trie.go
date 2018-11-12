package vm

import (
	"encoding/hex"

	"github.com/EducationEKT/EKT/MPTPlus"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
)

func builtin_awm_mpt_init(call FunctionCall) Value {
	mpt := MPTPlus.NewMTP(db.GetDBInst())
	return toValue_string(hex.EncodeToString(mpt.Root))
}

func builtin_awm_mpt_insert(call FunctionCall) Value {
	root, err := hex.DecodeString(call.Argument(0).string())
	key := call.Argument(1).string()
	value := call.Argument(2).string()
	if err != nil {
		return toValue_string("")
	}
	mpt := MPTPlus.MTP_Tree(db.GetDBInst(), root)
	if mpt.MustInsert(crypto.Sha3_256([]byte(key)), []byte(value)) != nil {
		return toValue_string("")
	}
	return toValue_string(hex.EncodeToString(mpt.Root))
}

func builtin_awm_mpt_get(call FunctionCall) Value {
	root, err := hex.DecodeString(call.Argument(0).string())
	key := crypto.Sha3_256([]byte(call.Argument(1).string()))
	if err != nil {
		return toValue_string("")
	}
	mpt := MPTPlus.MTP_Tree(db.GetDBInst(), root)

	value, err := mpt.GetValue(key)
	if err != nil {
		return toValue_string("")
	}
	return toValue_string(string(value))
}
