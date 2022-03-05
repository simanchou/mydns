package handler

import (
	"mydns/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"mydns/master/internal/logic"
	"mydns/master/internal/svc"
	"mydns/master/internal/types"
)

func RecordDelHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Zone
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewRecordDelLogic(r.Context(), ctx)
		resp, err := l.RecordDel(req)
		response.Response(w, resp, err)
	}
}
