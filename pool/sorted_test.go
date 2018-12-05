package pool

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
)

func TestSortedSave(t *testing.T) {
	from, _ := hex.DecodeString("56b92dfdbfbd7d32ea5deb6ca05ea8d695ed727c9d9a7536e345646608e339dc")
	pubKey, _ := crypto.GenerateKeyPair()
	to := hex.EncodeToString(types.FromPubKeyToAddress(pubKey))
	transactionOne := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 4, "test", " ")
	transactionTwo := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 3, "test", " ")
	transactionThree := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 2, "test", " ")
	transactionFour := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 4, "test", " ")
	transactionFive := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 5, "test", " ")
	transactionSix := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 6, "test", " ")
	transactionSeven := userevent.NewTransaction(from, []byte(to), time.Now().Unix(), 50, 50, 100, "test", " ")
	userTx := NewUserTxs(4)
	userTx.Save(transactionOne)
	fmt.Println("first save index", userTx.Index)
	userTx.Save(transactionTwo)
	fmt.Println("secend save index", userTx.Index)
	userTx.Save(transactionThree)
	fmt.Println("Three save index", userTx.Index)
	userTx.Save(transactionFour)
	fmt.Println("Four save index", userTx.Index)
	userTx.Save(transactionFive)
	fmt.Println("Five save index", userTx.Index)
	userTx.Save(transactionSix)
	fmt.Println("Six save index", userTx.Index)
	userTx.Save(transactionSeven)
	fmt.Println("Seven save index", userTx.Index)
}
