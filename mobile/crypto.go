package mobile

import (
	"encoding/hex"
	"strings"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

func Sign(msg, privKey []byte) ([]byte, error) {
	return crypto.Crypto(msg, privKey)
}

func SignMsg(data, privKey string) string {
	msg := Sha3_256([]byte(data))
	if strings.HasPrefix(privKey, "0x") {
		privKey = privKey[2:]
	}
	pk, _ := hex.DecodeString(privKey)
	sign, _ := Sign(msg, pk)
	return hex.EncodeToString(sign)
}

func RecoverPubKey(msg, sign []byte) ([]byte, error) {
	return crypto.RecoverPubKey(msg, sign)
}

func GetPubKeyFromPrivKey(priv []byte) ([]byte, error) {
	return crypto.PubKey(priv)
}

func PubKey2Address(pubKey []byte) []byte {
	return types.FromPubKeyToAddress(pubKey)
}

func PrivKey2Address(priv string) string {
	if strings.HasPrefix(priv, "0x") {
		priv = priv[2:]
	}
	privKey, _ := hex.DecodeString(priv)
	pub, _ := GetPubKeyFromPrivKey(privKey)
	return hex.EncodeToString(types.FromPubKeyToAddress(pub))
}

func GetAccount() string {
	pub, priv := crypto.GenerateKeyPair()
	address := PubKey2Address(pub)
	return hex.EncodeToString(address) + "_" + hex.EncodeToString(priv)
}

func Sha3_256(data []byte) []byte {
	return crypto.Sha3_256(data)
}
