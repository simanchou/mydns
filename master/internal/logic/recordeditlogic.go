package logic

import (
	"context"
	"fmt"
	"os/exec"

	"mydns/master/internal/svc"
	"mydns/master/internal/types"

	"github.com/tal-tech/go-zero/core/logx"
)

type RecordEditLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecordEditLogic(ctx context.Context, svcCtx *svc.ServiceContext) RecordEditLogic {
	return RecordEditLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecordEditLogic) RecordEdit(req types.Zone) (resp *types.BaseResp, err error) {
	var (
		out []byte
	)
	z := zone{}
	z.Domain = req.Domain
	z.Serial = req.Serial
	z.Records = req.Records
	err = genZoneFile(z, l.svcCtx)
	if err != nil {
		return nil, err
	}

	cmdStr := fmt.Sprintf("rndc reload %s", req.Domain)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err = cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("record %s add fail, msg: %s", req.Domain, out)
		return nil, err
	}
	logx.Infof("record %s add successful, %s", req.Domain, out)

	return &types.BaseResp{Ok: true}, nil
}
