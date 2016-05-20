package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"time"
)

func VerifySignature(signedText []byte, signature []byte) bool {
	mac := hmac.New(sha512.New, []byte(SharedSecret))
	mac.Write(signedText)
	expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal(signature, expectedMAC)
}

func VerifyTime(timestamp int64) (int64, error) {
	timeSinceRequest := time.Now().Unix() - timestamp
	if timeSinceRequest > 10 {
		return -1, errors.New("Timestamp is too far in the past")
	}
	return timestamp, nil
}
