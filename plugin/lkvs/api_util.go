package lkvs

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

const (
	SUCCESS             = 200
	ERROR               = 500
	INVALID_RECORD_TYPE = 300
	INVALID_PARAMS      = 400

	ERROR_EXIST_ZONE                              = 10001
	ERROR_EXIST_ZONE_FAIL                         = 10002
	ERROR_NOT_EXIST_ZONE                          = 10003
	ERROR_GET_ZONES_FAIL                          = 10004
	ERROR_COUNT_ZONE_FAIL                         = 10005
	ERROR_ADD_ZONE_FAIL                           = 10006
	ERROR_EDIT_ZONE_FAIL                          = 10007
	ERROR_DELETE_ZONE_FAIL                        = 10008
	ERROR_CAN_NOT_DELETE_ZONE_WHEN_RECORD_NOT_NIL = 10009
	ERROR_EXPORT_ZONE_FAIL                        = 10010
	ERROR_IMPORT_ZONE_FAIL                        = 10011

	ERROR_EXIST_RECORD       = 10021
	ERROR_EXIST_RECORD_FAIL  = 10022
	ERROR_NOT_EXIST_RECORD   = 10023
	ERROR_ADD_RECORD_FAIL    = 10024
	ERROR_DELETE_RECORD_FAIL = 10025
	ERROR_EDIT_RECORD_FAIL   = 10026
	ERROR_COUNT_RECORD_FAIL  = 10027
	ERROR_GET_RECORDS_FAIL   = 10028
	ERROR_GET_RECORD_FAIL    = 10029

	ERROR_AUTH_CHECK_TOKEN_FAIL    = 20001
	ERROR_AUTH_CHECK_TOKEN_TIMEOUT = 20002
	ERROR_AUTH_TOKEN               = 20003
	ERROR_AUTH                     = 20004
)

// MsgFlags flags of msg
var CodeMsgFlags = map[int]string{
	SUCCESS:                "ok",
	ERROR:                  "fail",
	INVALID_RECORD_TYPE:    "错误的记录类型",
	INVALID_PARAMS:         "请求参数错误",
	ERROR_EXIST_ZONE:       "已存在该域名",
	ERROR_EXIST_ZONE_FAIL:  "获取已存在域名失败",
	ERROR_NOT_EXIST_ZONE:   "该域名不存在",
	ERROR_GET_ZONES_FAIL:   "获取所有域名失败",
	ERROR_COUNT_ZONE_FAIL:  "统计域名失败",
	ERROR_ADD_ZONE_FAIL:    "新增域名失败",
	ERROR_EDIT_ZONE_FAIL:   "修改域名失败",
	ERROR_DELETE_ZONE_FAIL: "删除域名失败",
	ERROR_CAN_NOT_DELETE_ZONE_WHEN_RECORD_NOT_NIL: "还有记录时不能删除域名",
	ERROR_EXPORT_ZONE_FAIL:                        "导出域名失败",
	ERROR_IMPORT_ZONE_FAIL:                        "导入域名失败",
	ERROR_EXIST_RECORD:                            "该记录已存在",
	ERROR_NOT_EXIST_RECORD:                        "该记录不存在",
	ERROR_ADD_RECORD_FAIL:                         "新增记录失败",
	ERROR_DELETE_RECORD_FAIL:                      "删除记录失败",
	ERROR_EXIST_RECORD_FAIL:                       "检查已存在记录失败",
	ERROR_EDIT_RECORD_FAIL:                        "修改记录失败",
	ERROR_COUNT_RECORD_FAIL:                       "统计记录失败",
	ERROR_GET_RECORDS_FAIL:                        "获取多个记录失败",
	ERROR_GET_RECORD_FAIL:                         "获取单个记录失败",
	ERROR_AUTH_CHECK_TOKEN_FAIL:                   "Token鉴权失败",
	ERROR_AUTH_CHECK_TOKEN_TIMEOUT:                "Token已超时",
	ERROR_AUTH_TOKEN:                              "Token生成失败",
	ERROR_AUTH:                                    "Token错误",
}

// GetMsg get error msg of error code
func GetCodeMsg(code int) string {
	msg, ok := CodeMsgFlags[code]
	if ok {
		return msg
	}

	return CodeMsgFlags[ERROR]
}

type Gin struct {
	C *gin.Context
}

type ApiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (g *Gin) Response(httpCode, errCode int, data interface{}) {
	g.C.JSON(httpCode, ApiResponse{
		Code: errCode,
		Msg:  GetCodeMsg(errCode),
		Data: data,
	})
}

func GenerateRecordID() string {
	u4 := uuid.Must(uuid.NewV4(), nil)
	return fmt.Sprintf("%s", u4)
}
