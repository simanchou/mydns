package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"mydns/master/internal/config"
	"mydns/master/internal/handler"
	"mydns/master/internal/logic"
	"mydns/master/internal/middleware"
	"mydns/master/internal/svc"
	"mydns/utils"
	"net/http"
	"os"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/master-master.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)

	// get the API_KEY,API_SECRET from ENV
	middleware.ApiKey = os.Getenv("API_KEY")
	middleware.ApiSecret = os.Getenv("API_SECRET")
	if middleware.ApiKey == "" {
		middleware.ApiKey = utils.RandomString(36)
	}
	if middleware.ApiSecret == "" {
		middleware.ApiSecret = utils.RandomString(22)
	}
	logx.Infof("request header should have the author info like, key: \"Authorization\", value: \"%s\"", fmt.Sprintf("sso-key %s:%s", middleware.ApiKey, middleware.ApiSecret))

	// get the SLAVES from ENV
	logic.Slaves = strings.Split(os.Getenv("SLAVES"), ",")
	if len(logic.Slaves) == 0 {
		logx.Errorf("can not found any slave from key \"SLAVES\" in ENV, running in MASTER alone")
	} else {
		// get the SLAVE_PORT from ENV
		logic.SlavePort = os.Getenv("SLAVE_PORT")
		if logic.SlavePort == "" {
			logic.SlavePort = "52099"
		}
	}

	// init http client
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	server.Start()
}
