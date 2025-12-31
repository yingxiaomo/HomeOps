package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/yingxiaomo/homeops/config"
)

var (
	permMutex   sync.RWMutex
	permissions map[string][]string
	permFile    = "permissions.json"
)

func init() {
	LoadPermissions()
}

func LoadPermissions() {
	permMutex.Lock()
	defer permMutex.Unlock()

	permissions = make(map[string][]string)
	data, err := ioutil.ReadFile(permFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading permissions file: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, &permissions); err != nil {
		log.Printf("Error unmarshalling permissions: %v", err)
	}
}

func SavePermissions() {
	permMutex.Lock()
	defer permMutex.Unlock()

	data, err := json.MarshalIndent(permissions, "", "  ")
	if err != nil {
		log.Printf("Error marshalling permissions: %v", err)
		return
	}

	if err := ioutil.WriteFile(permFile, data, 0644); err != nil {
		log.Printf("Error writing permissions file: %v", err)
	}
}

func IsAdmin(userID int64) bool {
	return userID == config.AppConfig.AdminID
}

func HasPermission(userID int64, feature string) bool {
	if IsAdmin(userID) {
		return true
	}

	permMutex.RLock()
	defer permMutex.RUnlock()

	uidStr := strconv.FormatInt(userID, 10)
	features, ok := permissions[uidStr]
	if !ok {
		return false
	}

	if feature == "" {
		return true
	}

	for _, f := range features {
		if f == feature || f == "all" {
			return true
		}
	}

	return false
}

func GrantPermission(userIDStr string, feature string) bool {
	permMutex.Lock()
	defer permMutex.Unlock()

	features, ok := permissions[userIDStr]
	if !ok {
		features = []string{}
	}

	for _, f := range features {
		if f == feature {
			return false // Already has permission
		}
	}

	features = append(features, feature)
	permissions[userIDStr] = features
	return true
}

func RevokePermission(userIDStr string, feature string) bool {
	permMutex.Lock()
	defer permMutex.Unlock()

	features, ok := permissions[userIDStr]
	if !ok {
		return false
	}

	newFeatures := []string{}
	found := false
	for _, f := range features {
		if f == feature {
			found = true
			continue
		}
		newFeatures = append(newFeatures, f)
	}

	if !found {
		return false
	}

	if len(newFeatures) == 0 {
		delete(permissions, userIDStr)
	} else {
		permissions[userIDStr] = newFeatures
	}

	return true
}

func GetPermissions() map[string][]string {
	permMutex.RLock()
	defer permMutex.RUnlock()

	// Return copy
	copy := make(map[string][]string)
	for k, v := range permissions {
		copy[k] = v
	}
	return copy
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func EscapeMarkdown(text string) string {
	// Markdown V1 only requires escaping these characters
	special := "_*[]`"
	for _, c := range special {
		text = strings.ReplaceAll(text, string(c), "\\"+string(c))
	}
	return text
}
