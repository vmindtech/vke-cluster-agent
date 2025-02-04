package service

import (
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"time"

	"gorm.io/datatypes"
)

func Base64Encoder(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func GetRandomStringFromArray(a []string) string {
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(a))
	return a[i]
}

func IsValidBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func ConvertDataJSONtoStringArray(jsonData datatypes.JSON) []string {
	result := []string{}
	_ = json.Unmarshal([]byte(jsonData), &result)

	return result
}

func DeleteItemFromArray(a []string, item string) []string {
	for i, v := range a {
		if v == item {
			return append(a[:i], a[i+1:]...)
		}
	}
	return a
}
