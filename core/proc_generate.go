package core

import (
	"fmt"
	"math/rand"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}


func GenerateMessages(n int, priority int) []string {
	var msgs []string
	msg := randStringBytes()
	for i := 0; i < n; i++ {
		msgs = append(msgs, fmt.Sprintf("Message - %s with priority %d: #%d", msg, priority, i + 1))
	}
	return msgs
}