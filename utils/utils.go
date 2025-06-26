package utils

import (
	"math/rand"
	"strconv"
	"time"
)

func GenerateRandomID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(rand.Intn(10000))
}
