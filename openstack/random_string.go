package openstack

import (
	"crypto/rand"
	"math/big"
	mathRand "math/rand"
	"strings"
)

const (
	NameDefaultSet = `0123456789abcdefghijklmnopqrstuvwxyz`
	PwdDefaultSet  = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz~!@#$%^&*()_+-={}[]:"'<>,./|\'?`

	lowerCharSet   = "abcdefghijklmnopqrstuvwxyz"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = `~!@#$%^&*()_+-={}[]:\"'<>,./|\\'?`
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

func RandomString(charset string, size int) string {
	var bytes = make([]byte, size)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

func RandomStringWithPrefix(prefix, charset string, size int) string {
	var bytes = make([]byte, size)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return prefix + string(bytes)
}

func generatePassword(passwordLength, minSpecialChar, minNum, minUpperCase, minLowerCase int) (string, error) {
	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(specialCharSet))))
		if err != nil {
			return "", err
		}
		password.WriteString(string(specialCharSet[random.Int64()]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(numberSet))))
		if err != nil {
			return "", err
		}
		password.WriteString(string(numberSet[random.Int64()]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(upperCharSet))))
		if err != nil {
			return "", err
		}
		password.WriteString(string(upperCharSet[random.Int64()]))
	}

	//Set lowercase
	for i := 0; i < minLowerCase; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(lowerCharSet))))
		if err != nil {
			return "", err
		}
		password.WriteString(string(lowerCharSet[random.Int64()]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(allCharSet))))
		if err != nil {
			return "", err
		}
		password.WriteString(string(allCharSet[random.Int64()]))
	}

	runePwd := []rune(password.String())
	mathRand.Shuffle(len(runePwd), func(i, j int) {
		runePwd[i], runePwd[j] = runePwd[j], runePwd[i]
	})

	return string(runePwd), nil
}
