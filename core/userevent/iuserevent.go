package userevent

import (
	"bytes"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

func ValidateTransaction(transaction Transaction) bool {
	if bytes.Equal(transaction.GetFrom(), transaction.GetTo()) {
		return false
	}
	if transaction.Amount < 0 {
		return false
	}
	pubKey, err := crypto.RecoverPubKey(transaction.Msg(), transaction.GetSign())
	if err != nil {
		return false
	}
	result := bytes.Equal(types.FromPubKeyToAddress(pubKey), transaction.GetFrom())
	return result
}

func SignTransaction(transaction *Transaction, privKey []byte) error {
	sign, err := crypto.Crypto(transaction.Msg(), privKey)
	if err != nil {
		return err
	}
	transaction.SetSign(sign)
	return nil
}
