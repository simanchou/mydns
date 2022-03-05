package handler

import (
	"mydns/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"mydns/slave/internal/logic"
	"mydns/slave/internal/svc"
	"mydns/slave/internal/types"
)

func DomainAddHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewDomainAddLogic(r.Context(), ctx)
		resp, err := l.DomainAdd(req)
		response.Response(w, resp, err)
	}
}
