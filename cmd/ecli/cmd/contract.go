package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/cmd/ecli/param"
	"github.com/EducationEKT/EKT/ektclient"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"

	"github.com/spf13/cobra"
)

var ContractCmd *cobra.Command

func init() {
	ContractCmd = &cobra.Command{
		Use:   "contract",
		Short: "contract operate",
	}
	ContractCmd.AddCommand([]*cobra.Command{
		&cobra.Command{
			Use:   "deploy",
			Short: "deploy contract",
			Run:   DeployContract,
		},
	}...)
}

func DeployContract(cmd *cobra.Command, args []string) {
	fmt.Print("Please input your private key: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	privateKey := input.Text()
	if strings.HasPrefix(privateKey, "0x") {
		privateKey = privateKey[2:]
	}
	private, err := hex.DecodeString(privateKey)
	if err != nil || len(private) != 32 {
		fmt.Println("Your private key is wrong!")
		os.Exit(-1)
	}
	pub, _ := crypto.PubKey(private)
	address := types.FromPubKeyToAddress(pub)
	fmt.Print("Please input you contract file path: ")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	contractFilePath := input.Text()
	f, err := os.Open(contractFilePath)
	if err != nil {
		os.Exit(-1)
		fmt.Println("Error file path.")
	}
	contract, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("Read contract failed.", err)
		os.Exit(-1)
	}
	nonce := getAccountNonce(hex.EncodeToString(address))
	tx := userevent.NewTransaction(address, []byte(""), time.Now().UnixNano()/1e6, 0, 0, nonce, string(contract), "")
	userevent.SignTransaction(tx, private)
	client := ektclient.NewClient(param.GetPeers())
	err = client.SendTransaction(*tx)
	if err != nil {
		fmt.Println("Deploy contract error, ", err)
	} else {
		fmt.Println("Send transaction success.")
	}
}

//func CallContract(cmd *cobra.Command, args []string) {
//	fmt.Print("Please input your private key: ")
//	input := bufio.NewScanner(os.Stdin)
//	input.Scan()
//	privateKey := input.Text()
//	if strings.HasPrefix(privateKey, "0x") {
//		privateKey = privateKey[2:]
//	}
//	private, err := hex.DecodeString(privateKey)
//	if err != nil || len(private) != 32 {
//		fmt.Println("Your private key is wrong!")
//		os.Exit(-1)
//	}
//	pub, _ := crypto.PubKey(private)
//	address := types.FromPubKeyToAddress(pub)
//	fmt.Print("Please input contract address: ")
//	input = bufio.NewScanner(os.Stdin)
//	input.Scan()
//	contractAddr := input.Text()
//	nonce := getAccountNonce(hex.EncodeToString(address))
//	tx := userevent.NewTransaction(address, []byte(""), time.Now().UnixNano()/1e6, 0, 0, nonce, string(contract), "")
//	userevent.SignTransaction(tx, private)
//	client := ektclient.NewClient(param.GetPeers())
//	err = client.SendTransaction(*tx)
//	if err != nil {
//		fmt.Println("Deploy contract error, ", err)
//	} else {
//		fmt.Println("Send transaction success.")
//	}
//}
