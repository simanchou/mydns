package response

import (
	"github.com/tal-tech/go-zero/rest/httpx"
	"net/http"
	"strings"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Response(w http.ResponseWriter, resp interface{}, err error) {
	var body Body
	if err != nil {
		body.Code = -1
		body.Msg = strings.Replace(err.Error(), "rpc error: code = Unknown desc = ", "", 1)
	} else {
		body.Msg = "ok"
		body.Data = resp
	}
	httpx.OkJson(w, body)
}
