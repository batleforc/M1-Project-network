package utils

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
)

func HashByteArray(content []byte) string {
	hashObj := sha256.New()
	hashObj.Write(content)

	x := fmt.Sprintf("%x", hashObj.Sum(nil))
	fmt.Printf("\n%sa\n", x)
	return x
}

func GetByteArrayFromPath(path string) (content []byte, err error) {
	content, err = ioutil.ReadFile(path)
	if err != nil {
		return content, fmt.Errorf("Unable to open local file: %v", err)
	}
	return content, nil
}
