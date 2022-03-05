package logic

import (
	"context"
	"fmt"
	"os/exec"

	"mydns/slave/internal/svc"
	"mydns/slave/internal/types"

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

func (l *DomainDelLogic) DomainDel(req types.DelReq) (resp *types.BaseResp, err error) {
	cmdStr := fmt.Sprintf("rndc delzone %s", req.Name)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	logx.Infof("domain %s delete successful, %s", req.Name, out)

	return &types.BaseResp{Ok: true}, nil
}
