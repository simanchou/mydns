package svc

import (
	"github.com/zeromicro/go-zero/rest"
	"mydns/slave/internal/config"
	"mydns/slave/internal/middleware"
)

type ServiceContext struct {
	Config config.Config
	Author rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Author: middleware.NewAuthorMiddleware().Handle,
	}
}
