package lkvs

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type apiResponse struct {
	Code int
	Msg  string
	Data interface{}
}

func (lkvs *LKVS) APIStart() {
	s := &http.Server{
		Addr:              fmt.Sprintf(":%d", lkvs.APIPort),
		Handler:           lkvs.APIEngine,
		ReadHeaderTimeout: 60 * time.Second,
		WriteTimeout:      60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	err := s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}

func (lkvs *LKVS) InitRouter() {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	gin.SetMode("debug")
	engine.POST("/register", lkvs.register)
	engine.POST("/auth", lkvs.getAuth)
	engine.GET("/rsync", lkvs.rsync)
	api := engine.Group("/api")
	api.Use(JWT())
	{
		api.GET("/domain", lkvs.apiGetZones)
		api.POST("/domain", lkvs.apiAddZone)
		api.DELETE("/domain", lkvs.apiDeleteZone)

		api.GET("/record", lkvs.apiGetRecords)
		api.POST("/record", lkvs.apiAddRecord)
		api.PUT("/record", lkvs.apiEditRecord)
		api.DELETE("/record", lkvs.apiDeleteRecord)

		api.GET("/user", lkvs.apiGetUsers)
		api.PUT("/user", lkvs.apiEditUser)
		api.DELETE("/user", lkvs.apiDeleteUser)
	}

	lkvs.APIEngine = engine
}

// register
func (lkvs *LKVS) register(c *gin.Context) {
	g := Gin{C: c}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		log.Println("read body from request fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}
	_u := &User{}
	err = json.Unmarshal(body, _u)
	if err != nil {
		log.Println("parse json fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}

	valid := validation.Validation{}
	u := NewUser(_u.Username, _u.Password)
	ok, _ := valid.Valid(u)

	if ok {
		_, isExist := lkvs.UserIsExist(u.Username)
		if !isExist {
			err := lkvs.Save(BucketNameForUser, u)
			if err != nil {
				g.Response(http.StatusOK, ERROR_ADD_USER_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_EXIST_USER, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}

//getAuth
func (lkvs *LKVS) getAuth(c *gin.Context) {
	g := Gin{C: c}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		log.Println("read body from request fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}
	_u := &User{}
	err = json.Unmarshal(body, _u)
	if err != nil {
		log.Println("parse json fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}

	u := NewUser(_u.Username, _u.Password)

	log.Printf("%#v\n", u)
	data := make(map[string]interface{})
	code := INVALID_PARAMS

	_, isExist := lkvs.UserIsExist(u.Username)
	if isExist {
		isAuth := lkvs.CheckAuth(u)
		if isAuth {
			token, err := GenerateToke(u)
			if err != nil {
				code = ERROR_AUTH_TOKEN
			} else {
				data["token"] = token
				code = SUCCESS
			}
		} else {
			code = ERROR_AUTH_WRONG_PASSWORD
		}
	} else {
		code = ERROR_NOT_EXIST_USER
	}

	g.Response(http.StatusOK, code, data)
}

// get all zones
func (lkvs *LKVS) apiGetZones(c *gin.Context) {
	g := Gin{C: c}
	allZones, err := lkvs.GetAllZones()
	if err != nil {
		g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
		return
	}

	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	data := make(map[string]*Zone)
	if user == "admin" {
		data = allZones
	} else {
		for k, v := range allZones {
			if v.User == user {
				data[k] = v
			}
		}
	}

	g.Response(http.StatusOK, SUCCESS, data)
}

// add zone
func (lkvs *LKVS) apiAddZone(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	z := NewZone()
	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			data := validation.Error{
				Message: GetCodeMsg(ERROR_EXIST_ZONE),
				Key:     zoneName,
				Name:    zoneName,
				Value:   zoneName}
			g.Response(http.StatusOK, ERROR_EXIST_ZONE, data)
			return
		} else {
			z.Zone = zoneName
			z.User = user
			z.SOA.MBox = fmt.Sprintf("admin.%s", zoneName)
			z.SOA.Ns = "ns1.mydns.local."

			z.Records = make(map[string]*Record)
			nsRecord1 := NewRecord()
			nsRecord1.Type = "NS"
			nsRecord1.SubDomain = "@"
			nsRecord1.Host = "ns1.mydns.local."
			nsRecord2 := NewRecord()
			nsRecord2.Type = "NS"
			nsRecord2.SubDomain = "@"
			nsRecord2.Host = "ns2.mydns.local."

			z.Records[nsRecord1.ID] = nsRecord1
			z.Records[nsRecord2.ID] = nsRecord2
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}
	err := lkvs.Save(BucketNameForDomain, z)
	if err != nil {
		g.Response(http.StatusInternalServerError, ERROR_ADD_ZONE_FAIL, nil)
		return
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}

// delete zone
func (lkvs *LKVS) apiDeleteZone(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")

	userRecord := 0
	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}
		// 判断是否存在该域名
		if _z, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			// 判断域名是否属于该用户
			if _z.User == user || user == "admin" {
				// 判断域名的记录列表是否为空，不为空的话则不允许删除域名
				for _, r := range _z.Records {
					if r.Type != "NS" {
						userRecord += 1
					}
				}

			} else {
				g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}

	if userRecord == 0 {
		err := lkvs.DeleteZoneInDB(zoneName)
		if err != nil {
			g.Response(http.StatusInternalServerError, ERROR_DELETE_ZONE_FAIL, nil)
			return
		}
	} else {
		g.Response(http.StatusOK, ERROR_CAN_NOT_DELETE_ZONE_WHEN_RECORD_NOT_NIL, nil)
		return
	}

	g.Response(http.StatusOK, SUCCESS, nil)
}

// get all records of zone
func (lkvs *LKVS) apiGetRecords(c *gin.Context) {
	g := Gin{C: c}

	zoneName := AddDotAtLast(c.Query("domain"))
	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	if !valid.HasErrors() {

		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if data, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			if data.User == user || user == "admin" {
				g.Response(http.StatusOK, SUCCESS, data)
			} else {
				g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}
}

// add record
func (lkvs *LKVS) apiAddRecord(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")
	rType := c.Query("type")
	ttl := com.StrTo(c.DefaultQuery("ttl", "600")).MustInt()

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(rType, "type").Message("记录类型不能为空")

	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _z, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			if _z.User == user || user == "admin" {
				var z *Zone
				z = _z
				switch strings.ToUpper(rType) {
				case "A":
					code, err := AddARecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "AAAA":
					code, err := AddAAAARecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "TXT":
					code, err := AddTXTRecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "CNAME":
					code, err := AddCNAMERecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "MX":
					code, err := AddMXRecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "SRV":
					code, err := AddSRVRecordToZone(z, ttl, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				case "CAA":
					code, err := AddCAARecordToZone(z, c)
					if err != nil {
						g.Response(http.StatusOK, code, err)
						return
					}
				default:
					g.Response(http.StatusOK, INVALID_RECORD_TYPE, nil)
					return
				}

				err := lkvs.Save(BucketNameForDomain, z)
				if err != nil {
					g.Response(http.StatusInternalServerError, ERROR_ADD_ZONE_FAIL, nil)
					return
				}
			} else {
				g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}

	g.Response(http.StatusOK, SUCCESS, nil)
}

// edit record
func (lkvs *LKVS) apiEditRecord(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")
	id := c.Query("id")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(id, "id").Message("记录ID不能为空")

	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _z, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			if _z.User == user || user == "admin" {
				var z *Zone
				z = _z
				if r, ok := z.Records[id]; ok {
					switch r.Type {
					case "A":
						code, err := EditARecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "AAAA":
						code, err := EditAAAARecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "TXT":
						code, err := EditTXTRecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "CNAME":
						code, err := EditCNAMERecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "MX":
						code, err := EditMXRecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "SRV":
						code, err := EditSRVRecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}

					case "CAA":
						code, err := EditCAARecord(z, r, c)
						if err != nil {
							g.Response(http.StatusOK, code, err)
							return
						}
					default:
						g.Response(http.StatusOK, INVALID_RECORD_TYPE, nil)
						return
					}
				} else {
					g.Response(http.StatusOK, ERROR_NOT_EXIST_RECORD, nil)
					return
				}

				err := lkvs.Save(BucketNameForDomain, z)
				if err != nil {
					g.Response(http.StatusInternalServerError, ERROR_EDIT_RECORD_FAIL, nil)
					return
				}
			} else {
				g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}

	g.Response(http.StatusOK, SUCCESS, nil)
}

// delete record
func (lkvs *LKVS) apiDeleteRecord(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")
	id := c.Query("id")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(id, "id").Message("记录ID不能为空")

	var z *Zone
	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)

		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _z, ok, err := lkvs.ZoneIsExist(zoneName); ok {
			if err != nil {
				g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
				return
			}
			if _z.User == user || user == "admin" {
				z = _z
				if r, ok := z.Records[id]; ok {
					if r.Type != "NS" {
						delete(z.Records, id)
					} else {
						g.Response(http.StatusOK, ERROR_CAN_NOT_DELETE_NS_RECORD, nil)
						return
					}
				} else {
					g.Response(http.StatusOK, ERROR_NOT_EXIST_RECORD, nil)
					return
				}
			} else {
				g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
			return
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}

	err := lkvs.Save(BucketNameForDomain, z)
	if err != nil {
		g.Response(http.StatusInternalServerError, ERROR_DELETE_RECORD_FAIL, nil)
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}

// get user
func (lkvs *LKVS) apiGetUsers(c *gin.Context) {
	g := Gin{C: c}
	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	data := lkvs.GetAllUsers()
	if user == "admin" {
		g.Response(http.StatusOK, SUCCESS, data)
	} else {
		var _data []*User
		for _, i := range data {
			if i.Username == user {
				_data = append(_data, i)
			}
		}
		g.Response(http.StatusOK, SUCCESS, _data)
	}
}

// edit user
func (lkvs *LKVS) apiEditUser(c *gin.Context) {
	g := Gin{C: c}
	userFromToken, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}
	userFromForm := DeleteSpace(c.Query("user"))

	userForModify := ""
	switch {
	case userFromToken == "admin" && userFromForm != "":
		userForModify = userFromForm
	default:
		userForModify = userFromToken
	}

	u, isExist := lkvs.UserIsExist(userForModify)
	if isExist {
		oldPW := DeleteSpace(c.Query("old_password"))
		newPW := DeleteSpace(c.Query("new_password"))

		type pwCheck struct {
			OldPW string `valid:"Required; MaxSize(50)"`
			NewPW string `valid:"Required; MaxSize(50)"`
		}

		pc := pwCheck{OldPW: oldPW, NewPW: newPW}
		valid := validation.Validation{}
		ok, err := valid.Valid(pc)
		if err != nil {
			g.Response(http.StatusOK, ERROR_REQUIRE_CHECK_FAIL, err)
		}

		if ok {
			u.Password = EncryptionPassword(oldPW)
			isAuth := lkvs.CheckAuth(u)
			if isAuth {
				u.Password = EncryptionPassword(newPW)
				err := lkvs.Save(BucketNameForUser, u)
				if err != nil {
					g.Response(http.StatusOK, ERROR_EDIT_USER_FAIL, nil)
					return
				}
				g.Response(http.StatusOK, SUCCESS, nil)
			} else {
				g.Response(http.StatusOK, ERROR_OLD_PASSWORD_WRONG, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_REQUIRE_CHECK_FAIL,
				&validation.Error{
					Message: GetCodeMsg(ERROR_REQUIRE_CHECK_FAIL),
					Key:     "old_password or new_password",
					Name:    "old_password or new_password",
					Value:   "不能为空，并且长度不能超过50个字符",
				})
		}
	} else {
		g.Response(http.StatusOK, ERROR_NOT_EXIST_USER, nil)
		return
	}
}

// delete user
func (lkvs *LKVS) apiDeleteUser(c *gin.Context) {
	g := Gin{C: c}
	userFromToken, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	if userFromToken == "admin" {
		user := DeleteSpace(c.Query("user"))
		if user == "admin" {
			g.Response(http.StatusOK, ERROR_AUTH_ADMIN_CAN_NOT_DELETE, nil)
			return
		}
		_, isExist := lkvs.UserIsExist(user)
		if isExist {
			err := lkvs.DeleteUserInDB(user)
			if err != nil {
				g.Response(http.StatusOK, ERROR_DELETE_USER_FAIL, nil)
				return
			}
			g.Response(http.StatusOK, SUCCESS, nil)
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_USER, nil)
			return
		}

	} else {
		g.Response(http.StatusOK, ERROR_AUTH_ALLOW_ADMIN_ONLY, nil)
	}
}

// accept slave's rsync request
func (lkvs *LKVS) rsync(c *gin.Context) {
	g := Gin{C: c}

	slaveIP, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		g.Response(http.StatusOK, ERROR_GET_SLAVE_IP_FAIL, nil)
		return
	}

	if !lkvs.slaveIsAllow(slaveIP) {
		g.Response(http.StatusForbidden, ERROR_SLAVE_IS_NOT_ALLOW, nil)
		return
	}

	zones, err := lkvs.GetAllZones()
	if err != nil {
		g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
		return
	}

	g.Response(http.StatusOK, SUCCESS, zones)

}

func (lkvs *LKVS) slaveIsAllow(ip string) (ok bool) {
	allow := 0
	for _, i := range lkvs.Slave {
		if i == ip {
			allow++
			break
		}
	}
	if allow == 1 {
		ok = true
	} else {
		ok = false
	}
	return
}
