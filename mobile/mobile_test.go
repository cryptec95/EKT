package mobile

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCall(t *testing.T) {
	param := GoMobileParam{
		Method: "CreateAccount",
	}
	data, _ := json.Marshal(param)
	fmt.Println(Call(string(data)))
}
