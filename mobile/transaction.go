package mobile

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
)

func SendTrasaction(to, tokenAddr, data, privateKey string, amount int64) string {
	tx := buildTransaction(to, tokenAddr, data, privateKey, amount)
	if tx == nil {
		return buildResp(-400, map[string]interface{}{})
	}
	err := ektClient.SendTransaction(*tx)
	if err != nil {
		return buildResp(-500, map[string]interface{}{
			"err": err,
		})
	}
	return buildResp(0, map[string]interface{}{
		"txId": tx.TransactionId(),
	})
}

func BuildTransaction(to, tokenAddr, data, privateKey string, amount int64) string {
	tx := buildTransaction(to, tokenAddr, data, privateKey, amount)
	if tx == nil {
		return buildResp(-400, map[string]interface{}{})
	}
	return buildResp(0, map[string]interface{}{
		"tx": *tx,
	})
}

func buildTransaction(to, tokenAddr, data, privateKey string, amount int64) *userevent.Transaction {
	if strings.HasPrefix(privateKey, "0x") {
		privateKey = privateKey[2:]
	}
	private, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil
	}
	pubKey, err := crypto.PubKey(private)
	if err != nil {
		return nil
	}
	address := types.FromPubKeyToAddress(pubKey)

	if strings.HasPrefix(to, "0x") {
		to = to[2:]
	}
	toAddr, err := hex.DecodeString(to)
	if err != nil {
		return nil
	}

	nonce := ektClient.GetAccountNonce(hex.EncodeToString(address))
	fee := ektClient.GetSuggestionFee()
	timestamp := time.Now().UnixNano() / 1e6

	if amount < 0 {
		return nil
	}

	tx := userevent.NewTransaction(address, toAddr, timestamp, amount, fee, nonce, data, tokenAddr)
	err = userevent.SignTransaction(tx, private)
	if err != nil {
		return nil
	}
	return tx
}
