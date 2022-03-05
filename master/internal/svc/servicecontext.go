package svc

import (
	"github.com/zeromicro/go-zero/rest"
	"mydns/master/internal/config"
	"mydns/master/internal/middleware"
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
