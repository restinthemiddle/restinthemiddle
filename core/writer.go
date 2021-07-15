package core

import "net/http"

type Writer interface {
	LogRequest(request *http.Request) (err error)
	LogResponse(response *http.Response) (err error)
}
