package util

import "log"

func ErrorLog(err error, whatError string) {
	log.Printf("[ERROR] %s error: %s\n", whatError, err)
}

func InfoLog(msg string, data interface{}) {
	log.Printf("[INFO] %s %+v", msg, data)
}

func DebugLog(msg string, data interface{}) {
	log.Printf("[DEBUG] %s %+v", msg, data)
}
