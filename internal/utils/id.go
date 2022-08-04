package utils

import (
	"crypto/md5"
	"crypto/sha512"
	"fmt"
)

func getSha512Sum(resUrl string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(resUrl)))
}

func GetMd5Sum(resUrl string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(resUrl)))
}
