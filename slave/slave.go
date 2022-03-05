package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"mydns/slave/internal/middleware"
	"os"

	"mydns/slave/internal/config"
	"mydns/slave/internal/handler"
	"mydns/slave/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/slave-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)

	// get the master IP from ENV
	middleware.MasterIp = os.Getenv("MASTER_IP")
	if middleware.MasterIp == "" {
		middleware.MasterIp = "127.0.0.1"
	}
	logx.Infof("only allow the master \"%s\" to request the api, you can setup the value by key \"MASTER_IP\" in ENV", middleware.MasterIp)

	server.Start()
}
