package util

import "log"

func ErrorLog(err error, whatError string) {
	log.Printf("[ERROR] %s error: %s\n", whatError, err)
}
