package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"mydns/master/internal/svc"
	"mydns/master/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DomainDelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDomainDelLogic(ctx context.Context, svcCtx *svc.ServiceContext) DomainDelLogic {
	return DomainDelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DomainDelLogic) DomainDel(req types.Domain) (resp *types.BaseResp, err error) {
	cmdStr := fmt.Sprintf("rndc delzone %s ", req.Name)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	logx.Infof("domain %s delete successful, %s", req.Name, out)

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
				sReq       *http.Request
			)

			for _, slaveIp := range Slaves {
				slaveApi := fmt.Sprintf("https://%s:%s/domain/%s", slaveIp, SlavePort, req.Name)
				httpC := http.Client{}
				sReq, _ = http.NewRequest("DELETE", slaveApi, nil)
				sResp, err = httpC.Do(sReq)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Name, slaveIp, err.Error())
				}
				resp4Close = append(resp4Close, sResp)

				respBytes, err = ioutil.ReadAll(sResp.Body)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Name, slaveIp, err.Error())
				}
				slaveData := &slaveRespStruct{}
				err = json.Unmarshal(respBytes, slaveData)
				if err != nil {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Name, slaveIp, err.Error())
				}
				if slaveData.Code != 0 {
					logx.Errorf("domain %s sync to slave %s fail, %s", req.Name, slaveIp, slaveData.Msg)
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
