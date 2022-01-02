// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	"mydns/slave/internal/svc"

	"github.com/tal-tech/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Author},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/domain",
					Handler: DomainAddHandler(serverCtx),
				},
				{
					Method:  http.MethodDelete,
					Path:    "/domain/:name",
					Handler: DomainDelHandler(serverCtx),
				},
			}...,
		),
	)
}