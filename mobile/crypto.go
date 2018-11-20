package mobile

import (
	"encoding/hex"
	"strings"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

func SignMsg(data, privKey string) string {
	msg, _ := hex.DecodeString(Sha3_256(data))
	if strings.HasPrefix(privKey, "0x") {
		privKey = privKey[2:]
	}
	pk, _ := hex.DecodeString(privKey)
	sign, _ := crypto.Crypto(msg, pk)
	return hex.EncodeToString(sign)
}

func PrivateKey2Address(private string) string {
	if strings.HasPrefix(private, "0x") {
		private = private[2:]
	}
	privKey, _ := hex.DecodeString(private)
	pub, _ := crypto.PubKey(privKey)
	return hex.EncodeToString(types.FromPubKeyToAddress(pub))
}

func PubKey2Address(pub string) string {
	pubKey, _ := hex.DecodeString(pub)
	return hex.EncodeToString(types.FromPubKeyToAddress(pubKey))
}

func Sha3_256(data string) string {
	return hex.EncodeToString(crypto.Sha3_256([]byte(data)))
}
