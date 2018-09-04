package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/cmd/ecli/param"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/util"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

var TokenCommand *cobra.Command

func init() {
	TokenCommand = &cobra.Command{
		Use:   "token",
		Short: "Token Issue",
	}
	TokenCommand.AddCommand([]*cobra.Command{
		&cobra.Command{
			Use:   "issue",
			Short: "Public token.",
			Run:   IssueToken,
		},
	}...)
}

func IssueToken(cmd *cobra.Command, args []string) {
	fmt.Print("Input your token name: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	name := input.Text()

	fmt.Print("Input your token symbol: ")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	symbol := input.Text()

	fmt.Print("Input your token total amount: ")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	amount := input.Text()
	total, err := strconv.Atoi(amount)
	if err != nil {
		fmt.Println("Please input integer.")
		os.Exit(-1)
	}

	fmt.Print("Input your token decimals: ")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	decimal := input.Text()
	decimals, err := strconv.Atoi(decimal)
	if err != nil {
		fmt.Println("Please input integer.")
		os.Exit(-1)
	}

	fmt.Print("Input your private key: ")
	input = bufio.NewScanner(os.Stdin)
	input.Scan()
	priKey := input.Text()

	privKey := hexutil.MustDecode(priKey)
	pubKey, err := crypto.PubKey(privKey)
	if err != nil {
		fmt.Println("Invalid public private key")
		os.Exit(-1)
	}
	address := types.FromPubKeyToAddress(pubKey)

	token := types.Token{
		Name:     name,
		Symbol:   symbol,
		Total:    int64(total),
		Decimals: int64(decimals),
	}

	nonce := getAccountNonce(hex.EncodeToString(address))

	tokenIssue := &userevent.TokenIssue{
		Token:     token,
		From:      address,
		Nonce:     nonce,
		EventType: userevent.TYPE_USEREVENT_PUBLIC_TOKEN,
	}

	err = userevent.SignUserEvent(tokenIssue, privKey)
	if err != nil {
		os.Exit(-1)
	}
	SendTokenIssue(tokenIssue)
}

func SendTokenIssue(tokenIssue *userevent.TokenIssue) {
	for _, node := range param.GetPeers() {
		url := fmt.Sprintf(`http://%s:%d/token/api/issue`, node.Address, node.Port)
		_, err := util.HttpPost(url, tokenIssue.Bytes())
		if err == nil {
			break
		}
	}
}
