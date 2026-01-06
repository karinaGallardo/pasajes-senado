package appcontext

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	sessionMap = sync.Map{}
)

func SetAuthID(userID string) {
	sessionMap.Store(getGID(), userID)
}

func ClearAuthID() {
	sessionMap.Delete(getGID())
}

func AuthID() *string {
	if val, ok := sessionMap.Load(getGID()); ok {
		if id, ok := val.(string); ok && id != "" {
			return &id
		}
	}
	return nil
}

func getGID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseInt(idField, 10, 64)
	return id
}
