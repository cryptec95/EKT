package mobile

import (
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/crypto"
	"strings"
	"testing"
)

func TestCreateKeyStore(t *testing.T) {
	_, private := crypto.GenerateKeyPair()
	fmt.Println("PK:", hex.EncodeToString(private))
	keystore := CreateKeyStore(hex.EncodeToString(private), "12")
	fmt.Println("KeyStore:", keystore)
	decryptedKey := DecryptKeystore(keystore, "12")
	fmt.Println("DecryptedKey:", decryptedKey)
	if !strings.EqualFold(hex.EncodeToString(private), decryptedKey) {
		t.FailNow()
	}
}
