package lkvs

import (
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/scrypt"
	"net/http"
	"time"
)

const (
	SUCCESS             = 200
	ERROR               = 500
	INVALID_RECORD_TYPE = 300
	INVALID_PARAMS      = 400
	ERROR_REQUIRE_CHECK_FAIL = 600

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

	ERROR_ADD_USER_FAIL      = 20001
	ERROR_EXIST_USER         = 20002
	ERROR_NOT_EXIST_USER     = 20003
	ERROR_EDIT_USER_FAIL     = 20004
	ERROR_OLD_PASSWORD_WRONG = 20005
	ERROR_DELETE_USER_FAIL = 20006

	ERROR_AUTH_MISS_TOKEN = 30001
	ERROR_AUTH_CHECK_TOKEN_FAIL    = 30002
	ERROR_AUTH_CHECK_TOKEN_TIMEOUT = 30003
	ERROR_AUTH_TOKEN               = 30004
	ERROR_AUTH                     = 30005
	ERROR_AUTH_WRONG_PASSWORD = 30006
	ERROR_AUTH_ALLOW_ADMIN_ONLY = 30007
	ERROR_AUTH_ADMIN_CAN_NOT_DELETE = 30008
)

// MsgFlags flags of msg
var CodeMsgFlags = map[int]string{
	SUCCESS:                "ok",
	ERROR:                  "fail",
	INVALID_RECORD_TYPE:    "错误的记录类型",
	INVALID_PARAMS:         "请求参数错误",
	ERROR_REQUIRE_CHECK_FAIL:"条件限制检验失败",
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
	ERROR_DELETE_RECORD_FAIL:       "删除记录失败",
	ERROR_EXIST_RECORD_FAIL:        "检查已存在记录失败",
	ERROR_EDIT_RECORD_FAIL:         "修改记录失败",
	ERROR_COUNT_RECORD_FAIL:        "统计记录失败",
	ERROR_GET_RECORDS_FAIL:         "获取多个记录失败",
	ERROR_GET_RECORD_FAIL:          "获取单个记录失败",
	ERROR_ADD_USER_FAIL:            "注册用户失败",
	ERROR_EXIST_USER:               "用户已存在",
	ERROR_NOT_EXIST_USER:           "用户不存在",
	ERROR_EDIT_USER_FAIL:           "编辑用户失败",
	ERROR_OLD_PASSWORD_WRONG:       "旧密码错误",
	ERROR_DELETE_USER_FAIL:"删除用户失败",
	ERROR_AUTH_MISS_TOKEN:          "缺失token,访问此url需要先认证授权",
	ERROR_AUTH_CHECK_TOKEN_FAIL:    "Token鉴权失败",
	ERROR_AUTH_CHECK_TOKEN_TIMEOUT: "Token已超时",
	ERROR_AUTH_TOKEN:               "Token生成失败",
	ERROR_AUTH:                     "Token错误",
	ERROR_AUTH_WRONG_PASSWORD: "密码错误",
	ERROR_AUTH_ALLOW_ADMIN_ONLY:"仅允许超管执行该操作",
	ERROR_AUTH_ADMIN_CAN_NOT_DELETE:"超管不允许被删除",
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

// EncryptionPassword encryption password
var salt = []byte{5, 0, 7, 6, 9, 3, 9, 4}
func EncryptionPassword(pw string) (ep string) {
	dk, _ := scrypt.Key([]byte(pw), salt, 1<<15, 8, 1, 32)
	return base64.StdEncoding.EncodeToString(dk)
}

// jwt util
var jwtSecret = []byte("23347$040412")

// Claims claims
type Claims struct {
	Username string `json:"username"`
	Password string `json:"password"`
	jwt.StandardClaims
}

// GenerateToken generate token
func GenerateToke(u *User) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(3 * time.Hour)

	claims := Claims{
		Username: u.Username,
		Password: u.Password,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "mydns",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)

	return token, err
}

// ParseToken parse token
func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, err
}

// GetUserFromToken
func GetUserFromToken(c *gin.Context) (userName string, err error) {
	var claims *Claims
	token := c.Query("token")
	claims, err = ParseToken(token)
	if err != nil {
		return "", err
	}
	return claims.Username, nil
}

// JWT jwt middleware for gin
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		var code int
		var data interface{}

		code = SUCCESS
		token := c.Query("token")
		if token == "" {
			code = ERROR_AUTH_MISS_TOKEN
		} else {
			claims, err := ParseToken(token)
			if err != nil {
				code = ERROR_AUTH_CHECK_TOKEN_FAIL
			} else if time.Now().Unix() > claims.ExpiresAt {
				code = ERROR_AUTH_CHECK_TOKEN_TIMEOUT
			}
		}

		if code != SUCCESS {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": code,
				"msg":  GetCodeMsg(code),
				"data": data,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
