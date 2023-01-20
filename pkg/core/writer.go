package core

import "net/http"

// A Writer logs the relevant part of an enriched HTTP response.
type Writer interface {
	LogResponse(response *http.Response) (err error)
}
