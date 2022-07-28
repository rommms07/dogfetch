package utils

import (
	"crypto/sha512"
	"fmt"
)

func getSha512Sum(resUrl string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(resUrl)))
}
