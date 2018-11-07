package vm

import (
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
