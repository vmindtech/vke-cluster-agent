package service

import (
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"gorm.io/datatypes"
	"k8s.io/client-go/kubernetes"
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

func IsExpired(currentDate time.Time, date time.Time, maintenanceWindow time.Duration) bool {
	return currentDate.Add(maintenanceWindow).After(date)
}

func GetKubernetesVersion(client *kubernetes.Clientset) string {
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return "0.0.0"
	}
	return version.String()
}

func IsVersionGreaterOrEqual(version, compareWith string) bool {
	version = strings.TrimPrefix(version, "v")
	compareWith = strings.TrimPrefix(compareWith, "v")

	vParts := strings.Split(version, ".")
	cParts := strings.Split(compareWith, ".")

	for len(vParts) < 3 {
		vParts = append(vParts, "0")
	}
	for len(cParts) < 3 {
		cParts = append(cParts, "0")
	}

	if vParts[0] != cParts[0] {
		return vParts[0] > cParts[0]
	}

	if vParts[1] != cParts[1] {
		return vParts[1] > cParts[1]
	}

	return vParts[2] >= cParts[2]
}
