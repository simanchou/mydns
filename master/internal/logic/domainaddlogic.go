package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tal-tech/go-zero/core/logx"
	"io/ioutil"
	"mydns/master/internal/svc"
	"mydns/master/internal/types"
	"net/http"
	"os/exec"
	"time"
)

type DomainAddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDomainAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) DomainAddLogic {
	return DomainAddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

var (
	Slaves    []string
	SlavePort string
)

func (l *DomainAddLogic) DomainAdd(req types.Zone) (resp *types.BaseResp, err error) {
	fmt.Printf("%#v\n", req)
	z := zone{}
	z.Domain = req.Domain
	z.Serial = req.Serial
	z.Records = req.Records
	err = genZoneFile(z, l.svcCtx)
	if err != nil {
		return nil, err
	}

	// sync to bind
	cmdStr := fmt.Sprintf("rndc addzone %s '{type master; file \"%s.zone\";};'", req.Domain, req.Domain)
	cmd := exec.Command("sh", "-c", cmdStr)
	var out []byte
	out, err = cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("domain %s add fail, msg: %s", req.Domain, out)
		return nil, err
	}
	logx.Infof("domain %s add successful, %s", req.Domain, out)

	// sync to slave in goroutine
	if len(Slaves) > 0 {
		go func() {
			type slaveRespStruct struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
			}
			var (
				resp4Close []*http.Response
				sResp      *http.Response
				respBytes  []byte
			)

			for _, slaveIp := range Slaves {
				slaveApi := fmt.Sprintf("https://%s:%s/domain?name=%s", slaveIp, SlavePort, req.Domain)
				sResp, err = http.Get(slaveApi)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Domain, slaveIp, err.Error())
				}
				resp4Close = append(resp4Close, sResp)

				respBytes, err = ioutil.ReadAll(sResp.Body)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Domain, slaveIp, err.Error())
				}
				slaveData := &slaveRespStruct{}
				err = json.Unmarshal(respBytes, slaveData)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Domain, slaveIp, err.Error())
				}
				if slaveData.Code != 0 {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Domain, slaveIp, slaveData.Msg)
				}

				time.Sleep(time.Second * 2)
			}

			// close http request
			for _, i := range resp4Close {
				i.Body.Close()
			}
		}()
	}

	return &types.BaseResp{Ok: true}, nil
}
