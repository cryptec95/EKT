package mobile

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
	"strings"
)

type KeyStore struct {
	Address    string            `json:"address"`
	PrivateKey EncryptPrivateKey `json:"privateKey"`
}

type EncryptPrivateKey struct {
	Version     int    `json:"version"`
	EncryptData string `json:"encryptData"`
}

func CreateKeyStore(key, auth string) string {
	defer func() {
		recover()
	}()
	private, err := hex.DecodeString(key)
	if err != nil {
		return ""
	}

	pubKey, err := crypto.PubKey(private)
	if err != nil {
		return ""
	}
	address := hex.EncodeToString(types.FromPubKeyToAddress(pubKey))
	encryptedKey, err := encryptPKV1(private, crypto.Sha3_256([]byte(auth))[:16])
	if err != nil {
		return ""
	}
	keystore := KeyStore{
		Address:    address,
		PrivateKey: *encryptedKey,
	}
	data, _ := json.Marshal(keystore)
	return string(data)
}

func DecryptKeystore(keystore, auth string) string {
	defer func() {
		recover()
	}()
	result := decryptPKV1(keystore, auth)
	if result != nil {
		return hex.EncodeToString(result)
	}
	return ""
}

func encryptPKV1(key, auth []byte) (*EncryptPrivateKey, error) {
	result, err := crypto.AesEncrypt(key, auth)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	privateKey := EncryptPrivateKey{
		Version:     1,
		EncryptData: base64.StdEncoding.EncodeToString(result),
	}
	return &privateKey, nil
}

func decryptPKV1(encrypted, auth string) []byte {
	data := []byte(encrypted)
	password := crypto.Sha3_256([]byte(auth))[:16]
	var key KeyStore
	err := json.Unmarshal(data, &key)
	if err != nil {
		return nil
	}
	if key.PrivateKey.Version != 1 {
		return nil
	}

	data, err = base64.StdEncoding.DecodeString(key.PrivateKey.EncryptData)
	if err != nil {
		return nil
	}

	result, err := crypto.AesDecrypt(data, password)
	if err != nil {
		return nil
	}
	pubKey, err := crypto.PubKey(result)
	if err != nil {
		return nil
	}
	address := hex.EncodeToString(types.FromPubKeyToAddress(pubKey))
	if !strings.EqualFold(address, key.Address) {
		return nil
	}

	return result
}
