package helpers

import (
	"crypto/rand"
	"fmt"
	"time"
)

func GenerateULID() string {
	return currentTimestamp() + generateRandomStr()
}

func currentTimestamp() string {
	timestamp := time.Now().UnixMicro()
	return fmt.Sprintf("%d", timestamp)
}

func generateRandomStr() string {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	const length = 10
	randomBytes := make([]byte, length)
	_, _ = rand.Read(randomBytes)
	for i := range randomBytes {
		randomBytes[i] = chars[randomBytes[i]%byte(26)]
	}
	return string(randomBytes)
}
