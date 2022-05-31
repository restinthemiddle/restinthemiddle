package core

import "net/http"

type Writer interface {
	LogResponse(response *http.Response) (err error)
}
