package handler

import (
	"mydns/response"
	"net/http"

	"github.com/tal-tech/go-zero/rest/httpx"
	"mydns/master/internal/logic"
	"mydns/master/internal/svc"
	"mydns/master/internal/types"
)

func DomainDelHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Domain
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewDomainDelLogic(r.Context(), ctx)
		resp, err := l.DomainDel(req)
		response.Response(w, resp, err)
	}
}
