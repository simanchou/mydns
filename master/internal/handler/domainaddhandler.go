package handler

import (
	"mydns/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"mydns/master/internal/logic"
	"mydns/master/internal/svc"
	"mydns/master/internal/types"
)

func DomainAddHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Zone
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewDomainAddLogic(r.Context(), ctx)
		resp, err := l.DomainAdd(req)
		response.Response(w, resp, err)
	}
}
