package lkvs

import (
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
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

	s.ListenAndServe()
}

func (lkvs *LKVS) InitRouter() {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	gin.SetMode("debug")
	engine.POST("/register", lkvs.register)
	engine.GET("/auth", lkvs.GetAuth)
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
	g := Gin{C:c}
	username := DeleteSpace(c.Query("username"))
	password := DeleteSpace(c.Query("password"))

	valid := validation.Validation{}
	u := NewUser(username, password)
	ok, _ := valid.Valid(u)

	if ok {
		_, isExist := lkvs.UserIsExist(u.Username)
		if ! isExist {
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

//GetAuth
func (lkvs *LKVS) GetAuth(c *gin.Context) {
	g := Gin{C:c}
	username := DeleteSpace(c.Query("username"))
	password := DeleteSpace(c.Query("password"))

	valid := validation.Validation{}
	u := NewUser(username, password)
	ok, err := valid.Valid(u)
	if err != nil {
		g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	data := make(map[string]interface{})
	code := INVALID_PARAMS

	if ok {
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
	}

	g.Response(http.StatusOK, code, data)
}

// get all zones
func (lkvs *LKVS) apiGetZones(c *gin.Context) {
	g := Gin{C: c}
	lkvs.LoadZones()

	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL,nil)
		return
	}

	data := make(map[string]Zone)
	if user == "admin" {
		data = lkvs.ZonesWithRecords
	} else {
		for k, v := range lkvs.ZonesWithRecords {
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
			g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL,nil)
			return
		}

		lkvs.LoadZones()

		if _, ok := lkvs.ZonesWithRecords[zoneName]; ok {
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
			z.SOA.Ns = "ns.mydns.local."
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

	if !valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL,nil)
			return
		}
		// 判断是否存在该域名
		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			// 判断域名是否属于该用户
			if _z.User == user || user == "admin" {
				// 判断域名的记录列表是否为空，不为空的话则不允许删除域名
				if _z.Records == nil {
					delete(lkvs.ZonesWithRecords, zoneName)
				} else {
					g.Response(http.StatusOK, ERROR_CAN_NOT_DELETE_ZONE_WHEN_RECORD_NOT_NIL, nil)
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

	err := lkvs.DeleteZoneInDB(zoneName)
	if err != nil {
		g.Response(http.StatusInternalServerError, ERROR_DELETE_ZONE_FAIL, nil)
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
		lkvs.LoadZones()

		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if data, ok := lkvs.ZonesWithRecords[zoneName]; ok {
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
		lkvs.LoadZones()

		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			if _z.User == user || user == "admin" {
				var z *Zone
				z = &_z
				if z.Records == nil {
					z.Records = make(map[string]*Record)
				}
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
		lkvs.LoadZones()

		user, err := GetUserFromToken(c)
		if err != nil {
			g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
			return
		}

		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			if _z.User == user || user == "admin" {
				var z *Zone
				z = &_z
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

		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			if _z.User == user || user == "admin" {
				z = &_z
				if _, ok := z.Records[id]; ok {
					delete(z.Records, id)
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
	g := Gin{C:c}
	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	data := lkvs.GetAllUsers()
	if user == "admin" {
		g.Response(http.StatusOK, SUCCESS, data)
	} else {
		_data := make(map[string]User)
		for k, v := range data {
			if v.Username == user {
				_data[k] = v
			}
		}
		g.Response(http.StatusOK, SUCCESS,_data)
	}
}

// edit user
func (lkvs *LKVS) apiEditUser(c *gin.Context) {
	g := Gin{C:c}
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

		pc := pwCheck{OldPW:oldPW, NewPW:newPW}
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
				g.Response(http.StatusOK, SUCCESS,nil)
			} else {
				g.Response(http.StatusOK,ERROR_OLD_PASSWORD_WRONG,nil)
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
	g := Gin{C:c}
	userFromToken, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	if userFromToken == "admin" {
		user := DeleteSpace(c.Query("user"))
		if user == "admin" {
			g.Response(http.StatusOK, ERROR_AUTH_ADMIN_CAN_NOT_DELETE,nil)
			return
		}
		_, isExist := lkvs.UserIsExist(user)
		if isExist {
			err := lkvs.DeleteUserInDB(user)
			if err != nil {
				g.Response(http.StatusOK, ERROR_DELETE_USER_FAIL,nil)
				return
			}
			g.Response(http.StatusOK,SUCCESS,nil)
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_USER, nil)
			return
		}

	} else {
		g.Response(http.StatusOK, ERROR_AUTH_ALLOW_ADMIN_ONLY, nil)
	}
}