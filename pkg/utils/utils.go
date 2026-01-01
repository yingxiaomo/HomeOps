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
	tele "gopkg.in/telebot.v3"
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
	special := "_*[]`"
	for _, c := range special {
		text = strings.ReplaceAll(text, string(c), "\\"+string(c))
	}
	return text
}

// SendLongMessage handles splitting long text into multiple messages.
// If msgToEdit is provided, the first chunk will edit that message.
// The menu will only be attached to the last chunk.
func SendLongMessage(c tele.Context, msgToEdit *tele.Message, text string, menu *tele.ReplyMarkup) error {
	const maxLen = 3800 // Leave some buffer for markdown overhead

	// Helper function to send or edit with fallback
	sendOrEdit := func(msg *tele.Message, txt string, opts ...interface{}) error {
		// First try with provided options (Markdown)
		var err error
		if msg != nil {
			_, err = c.Bot().Edit(msg, txt, opts...)
		} else {
			_, err = c.Bot().Send(c.Recipient(), txt, opts...)
		}

		// If successful, return
		if err == nil {
			return nil
		}

		// If error is about parsing, try plain text
		// We filter options to remove parsing modes
		var plainOpts []interface{}
		for _, opt := range opts {
			if _, ok := opt.(tele.ParseMode); !ok {
				plainOpts = append(plainOpts, opt)
			}
		}

		if msg != nil {
			_, err = c.Bot().Edit(msg, txt, plainOpts...)
		} else {
			_, err = c.Bot().Send(c.Recipient(), txt, plainOpts...)
		}
		return err
	}

	if len(text) <= maxLen {
		var opts []interface{}
		opts = append(opts, tele.ModeMarkdown)
		if menu != nil {
			opts = append(opts, menu)
		}
		return sendOrEdit(msgToEdit, text, opts...)
	}

	chunks := SplitText(text, maxLen)
	for i, chunk := range chunks {
		var opts []interface{}
		opts = append(opts, tele.ModeMarkdown)
		// Only attach menu to the last chunk
		if i == len(chunks)-1 && menu != nil {
			opts = append(opts, menu)
		}

		// For chunks, only the first one can be an Edit
		targetMsg := msgToEdit
		if i > 0 {
			targetMsg = nil // Subsequent chunks are always new messages
		}

		err := sendOrEdit(targetMsg, chunk, opts...)
		if err != nil {
			// If edit fails specifically (e.g. message too old/deleted), 
			// sendOrEdit might have already retried as plain text and failed,
			// or we might want to fallback to Send if Edit is impossible.
			// Simple fallback: if failure was on Edit, try Send (as plain text to be safe or retry logic)
			if targetMsg != nil {
				// Fallback to sending as a new message if editing failed completely
				sendOrEdit(nil, chunk, opts...)
			} else {
				return err
			}
		}
	}
	return nil
}

func SplitText(text string, limit int) []string {
	var chunks []string
	lines := strings.Split(text, "\n")
	var currentChunk strings.Builder

	for _, line := range lines {
		// +1 for newline
		if currentChunk.Len()+len(line)+1 > limit {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			// If the single line itself is too long, we must split it brutally
			if len(line) > limit {
				runes := []rune(line)
				for len(runes) > 0 {
					take := limit
					if len(runes) < take {
						take = len(runes)
					}
					chunks = append(chunks, string(runes[:take]))
					runes = runes[take:]
				}
			} else {
				currentChunk.WriteString(line)
				currentChunk.WriteString("\n")
			}
		} else {
			currentChunk.WriteString(line)
			currentChunk.WriteString("\n")
		}
	}
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}
