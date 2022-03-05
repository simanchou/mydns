package logic

import (
	"context"
	"fmt"
	"mydns/slave/internal/middleware"
	"os/exec"

	"mydns/slave/internal/svc"
	"mydns/slave/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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

func (l *DomainAddLogic) DomainAdd(req types.AddReq) (resp *types.BaseResp, err error) {
	cmdStr := fmt.Sprintf("rndc addzone %s \"{type slave; masters { %s; }; };\"", req.Name, middleware.MasterIp)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	logx.Infof("domain %s add successful, %s", req.Name, out)

	return &types.BaseResp{Ok: true}, nil
}
