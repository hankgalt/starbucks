package constants

import "time"

type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}

const SERVICE_PORT = 8080
const HEALTH_CHECK_URL = "/health"
const SEARCH_URL = "/search"

const READ_RATE = 500 * time.Millisecond
const ReadRateContextKey = ContextKey("readrate")

const FileNameContextKey = ContextKey("filename")
