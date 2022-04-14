package tools

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/satori/go.uuid"
)

func GetUUID() string {
	return fmt.Sprintf("%s", uuid.NewV4())
}

func GenerateUid() (string, error) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}