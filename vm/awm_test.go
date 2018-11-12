package vm

import (
	"github.com/EducationEKT/EKT/db"
	"testing"

	"encoding/hex"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

func TestRun(t *testing.T) {
	pub, pri := crypto.GenerateKeyPair()
	address := types.FromPubKeyToAddress(pub)
	msg := crypto.Sha3_256([]byte("123"))
	sign, _ := crypto.Crypto(msg, pri)
	vm := New()
	vm.Set("msg", hex.EncodeToString(msg))
	vm.Set("sign", hex.EncodeToString(sign))
	vm.Set("address", hex.EncodeToString(address))
	vm.Run(`
		console.log(AWM.sha3_256("123456"));
		console.log(AWM.verify(msg, sign, address));
		console.log(AWM.ecrecover(msg, sign));
		

	`)
}

func Test_AWM_MPT(t *testing.T) {
	if db.GetDBInst() == nil {
		db.InitEKTDB("test/testVM")
	}

	vm := New()

	vm.Run(`
		var root = AWM.mpt_init()
		console.log(root)
		root = AWM.mpt_save(root, "Hello", "world")
		console.log(root)
		console.log(AWM.mpt_get(root, "Hello"))
	`)

}
