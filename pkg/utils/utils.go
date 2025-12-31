package utils

import (
	"github.com/yingxiaomo/HomeOps/config"
	"math/rand"
)

func IsAdmin(userID int64) bool {
	return userID == config.AppConfig.AdminID
}

func HasPermission(userID int64, permission string) bool {
	if IsAdmin(userID) {
		return true
	}
	return false
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
