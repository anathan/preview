package util

import (
	"github.com/streadway/simpleuuid"
	"time"
)

var (
	NodeBytes = []byte{0x0f, 0xd1, 0x98, 0x10, 0x86, 0x2c, 0xd6, 0xdc}
)

func NewUuid() (string, error) {
	uuid, err := simpleuuid.NewTimeBytes(time.Now(), NodeBytes)
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}
