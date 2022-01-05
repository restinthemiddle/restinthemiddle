package core

import "net/http"

// Writer defines the interface for logging responses.
type Writer interface {
	LogResponse(response *http.Response) error
}
