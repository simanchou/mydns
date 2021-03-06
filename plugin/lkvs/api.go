package lkvs

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
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
		logger.Fatalln(err)
	}
}

func (lkvs *LKVS) InitRouter() {
	runMode := os.Getenv("DEBUG_ON")
	if runMode != "" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	engine.POST("/admin/register", lkvs.register)
	engine.POST("/admin/auth", lkvs.getAuth)
	engine.GET("/admin/rsync", lkvs.rsync)
	api := engine.Group("/admin/api")
	api.Use(JWT())
	{
		api.GET("/domains", lkvs.apiGetZones)
		api.POST("/domains", lkvs.apiAddZone)
		api.DELETE("/domains/:zone", lkvs.apiDeleteZone)

		api.GET("/record", lkvs.apiGetRecords)
		api.POST("/record", lkvs.apiAddRecordV2)
		api.PUT("/record", lkvs.apiEditRecordV2)
		api.DELETE("/record", lkvs.apiDeleteRecord)
		api.POST("/record/batch", lkvs.apiBatchRecord)

		api.POST("/logout", lkvs.logout)
		api.GET("/users", lkvs.apiGetUsers)
		api.GET("/users/info", lkvs.apiGetUserInfoWithoutID)
		api.POST("/users", lkvs.register)
		api.PUT("/users", lkvs.apiEditUser)
		api.POST("/users/chpw", lkvs.apiChangePW)
		api.DELETE("/users/:id", lkvs.apiDeleteUser)

		api.GET("/sys/sum", lkvs.apiGetSummary)
	}

	lkvs.APIEngine = engine
}

// register
func (lkvs *LKVS) register(c *gin.Context) {
	g := Gin{C: c}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		logger.Error("read body from request fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}
	_u := &User{}
	err = json.Unmarshal(body, _u)
	if err != nil {
		logger.Error("parse json fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}

	valid := validation.Validation{}
	u := NewUser(_u.Username, _u.Password)
	u.Nickname = _u.Nickname
	u.Roles = _u.Roles
	if _u.Avatar != "" {
		u.Avatar = _u.Avatar
	}
	ok, _ := valid.Valid(u)

	if ok {
		_, isExist := lkvs.UserIsExistByName(u.Username)
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
		logger.Error("read body from request fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}
	_u := &User{}
	err = json.Unmarshal(body, _u)
	if err != nil {
		logger.Error("parse json fail, ", err.Error())
		g.Response(http.StatusOK, INVALID_PARAMS, nil)
		return
	}

	u := NewUser(_u.Username, _u.Password)

	data := make(map[string]interface{})
	code := INVALID_PARAMS

	user, isExist := lkvs.UserIsExistByName(u.Username)
	if isExist {
		if user.Active {
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
			code = ERROR_USER_INACTIVE
		}

	} else {
		code = ERROR_NOT_EXIST_USER
	}

	g.Response(http.StatusOK, code, data)
}

// logout
func (lkvs *LKVS) logout(c *gin.Context) {
	g := Gin{C: c}
	g.Response(http.StatusOK, SUCCESS, nil)
}

// get user info
func (lkvs *LKVS) apiGetUserInfoWithoutID(c *gin.Context) {
	g := Gin{C: c}
	username, err := GetUserFromToken(c)
	if err != nil {
		logger.Error("get user from token fail, %s", err.Error())
		g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	u, _ := lkvs.UserIsExistByName(username)
	g.Response(http.StatusOK, SUCCESS, u)
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

	var data []*Zone
	if user == "admin" {
		for _, v := range allZones {
			data = append(data, v)
		}
	} else {
		for _, v := range allZones {
			if v.User == user {
				data = append(data, v)
			}
		}
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].CreateAt.After(data[j].CreateAt)
	})

	g.Response(http.StatusOK, SUCCESS, data)
}

// add zone
func (lkvs *LKVS) apiAddZone(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("zone")

	valid := validation.Validation{}
	valid.Required(zoneName, "zone").Message("??????????????????")
	z := NewZone()
	if !valid.HasErrors() {

		// get login user
		user, err := GetUserFromToken(c)

		// add public domain is allow by admin only
		isPublic := IsPublicDomain(zoneName)
		if isPublic {
			if user != "admin" {
				g.Response(http.StatusOK, ERROR_ZONE_IS_PUBLIC, fmt.Sprintf("%s is a public zone", zoneName))
				return
			}
		}

		zoneName = AddDotAtLast(zoneName)
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
			nsRecord1.Name = "@"
			nsRecord1.Value = "ns1.mydns.local."
			nsRecord2 := NewRecord()
			nsRecord2.Type = "NS"
			nsRecord2.Name = "@"
			nsRecord2.Value = "ns2.mydns.local."

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
	zoneName := c.Param("zone")

	userRecord := 0

	zoneName = AddDotAtLast(zoneName)
	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusOK, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}
	// ???????????????????????????
	if _z, ok, err := lkvs.ZoneIsExist(zoneName); ok {
		if err != nil {
			g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
			return
		}
		// ?????????????????????????????????
		if _z.User == user || user == "admin" {
			// ?????????????????????????????????????????????????????????????????????????????????
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
	valid.Required(zoneName, "domain").Message("??????????????????")
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
				var _data []*Record
				for _, v := range data.Records {
					_data = append(_data, v)
					sort.Slice(_data, func(i, j int) bool {
						if _data[i].Type > _data[j].Type {
							return false
						}
						if _data[i].Type < _data[j].Type {
							return true
						}
						if _data[i].Name > _data[j].Name {
							return false
						}
						if _data[i].Name < _data[j].Name {
							return true
						}
						return _data[i].Value < _data[j].Value
					})
				}
				g.Response(http.StatusOK, SUCCESS, _data)
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

// add record v2
func (lkvs *LKVS) apiAddRecordV2(c *gin.Context) {
	g := Gin{C: c}

	type record struct {
		Domain   string `json:"domain"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		Value    string `json:"value"`
		Priority uint16 `json:"priority"`
		TTL      uint32 `json:"ttl"`
	}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}

	r := &record{}
	err = json.Unmarshal(body, r)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}

	user, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}
	if z, ok, err := lkvs.ZoneIsExist(AddDotAtLast(r.Domain)); ok {
		if err != nil {
			g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
			return
		}

		recordIsExist := false
		for _, _record := range z.Records {
			if _record.Type == strings.ToUpper(r.Type) && _record.Name == r.Name && _record.Value == r.Value {
				recordIsExist = true
				g.Response(http.StatusOK, ERROR_EXIST_RECORD, nil)
				return
			}
		}

		_r := NewRecord()
		if z.User == user || user == "admin" {
			_r.Type = strings.ToUpper(r.Type)
			_r.Name = r.Name
			_r.Priority = r.Priority
			_r.TTL = r.TTL
			switch {
			case _r.Type == "A":
				if ip := net.ParseIP(r.Value); ip != nil {
					_r.Value = r.Value
				} else {
					g.Response(http.StatusOK, ERROR_INVALID_IP, nil)
					return
				}
			case _r.Type == "CNAME":
				_r.Value = AddDotAtLast(r.Value)
			case _r.Type == "MX":
				_r.Value = AddDotAtLast(r.Value)
			default:
				_r.Value = r.Value
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
			err = lkvs.Save(BucketNameForDomain, z)
			if err != nil {
				g.Response(http.StatusOK, ERROR, err.Error())
				return
			}
		}
		g.Response(http.StatusOK, SUCCESS, nil)
	} else {
		g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
		return
	}
}

// edit record v2
func (lkvs *LKVS) apiEditRecordV2(c *gin.Context) {
	g := Gin{C: c}

	type record struct {
		Domain   string `json:"domain"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		Value    string `json:"value"`
		Priority uint16 `json:"priority"`
		TTL      uint32 `json:"ttl"`
	}

	r := &record{}
	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		g.Response(http.StatusOK, ERROR, nil)
		return
	}
	err = json.Unmarshal(body, r)
	if err != nil {
		g.Response(http.StatusOK, ERROR, nil)
		return
	}

	if z, ok, err := lkvs.ZoneIsExist(AddDotAtLast(r.Domain)); ok && err == nil {
		_r := z.Records[r.ID]
		_r.Type = r.Type
		_r.Name = r.Name
		_r.Value = r.Value
		_r.Priority = r.Priority
		_r.TTL = r.TTL

		z.Records[r.ID] = _r
		err = lkvs.Save(BucketNameForDomain, z)
		if err != nil {
			g.Response(http.StatusInternalServerError, ERROR_EDIT_RECORD_FAIL, nil)
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
	valid.Required(zoneName, "domain").Message("??????????????????")
	valid.Required(id, "id").Message("??????ID????????????")

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

// batch delete record
func (lkvs *LKVS) apiBatchRecord(c *gin.Context) {
	g := Gin{C: c}

	type batchInfo struct {
		Domain string    `json:"domain"`
		Action string    `json:"action"`
		Items  []*Record `json:"items"`
	}

	items := &batchInfo{}
	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		g.Response(http.StatusOK, ERROR, nil)
		return
	}

	err = json.Unmarshal(body, items)
	if z, ok, err := lkvs.ZoneIsExist(AddDotAtLast(items.Domain)); ok && err == nil {
		for _, i := range items.Items {
			delete(z.Records, i.ID)
		}

		err := lkvs.Save(BucketNameForDomain, z)
		if err != nil {
			g.Response(http.StatusInternalServerError, ERROR_DELETE_RECORD_FAIL, nil)
		}
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

// user change password
func (lkvs *LKVS) apiChangePW(c *gin.Context) {
	g := Gin{C: c}

	userFromToken, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}
	u, _ := lkvs.UserIsExistByName(userFromToken)

	type pw struct {
		OldPW string `json:"oldPW"`
		NewPW string `json:"newPW"`
	}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}

	p := &pw{}
	err = json.Unmarshal(body, p)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}

	if EncryptionPassword(p.OldPW) != u.Password {
		g.Response(http.StatusOK, ERROR_OLD_PASSWORD_WRONG, nil)
		return
	}
	u.Password = EncryptionPassword(p.NewPW)
	err = lkvs.Save(BucketNameForUser, u)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}

// edit user
func (lkvs *LKVS) apiEditUser(c *gin.Context) {
	g := Gin{C: c}
	userFromToken, err := GetUserFromToken(c)
	if err != nil {
		g.Response(http.StatusUnauthorized, ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	if userFromToken != "admin" {
		g.Response(http.StatusOK, ERROR_AUTH_ALLOW_ADMIN_ONLY, nil)
		return
	}

	body, err := ioutil.ReadAll(g.C.Request.Body)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}
	_u := &User{}
	err = json.Unmarshal(body, _u)
	if err != nil {
		g.Response(http.StatusOK, ERROR, err.Error())
		return
	}

	if u, ok := lkvs.UserIsExistByName(_u.Username); ok {
		u.Avatar = _u.Avatar
		u.Nickname = _u.Nickname
		u.Roles = _u.Roles
		u.Active = _u.Active
		if _u.Password != "" {
			u.Password = EncryptionPassword(_u.Password)
		}
		err = lkvs.Save(BucketNameForUser, u)
		if err != nil {
			g.Response(http.StatusOK, ERROR_EDIT_USER_FAIL, err.Error())
			return
		}
	} else {
		g.Response(http.StatusOK, ERROR_NOT_EXIST_USER, nil)
		return
	}

	g.Response(http.StatusOK, SUCCESS, nil)
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
		userId := DeleteSpace(c.Param("id"))
		user, isExist := lkvs.UserIsExistById(userId)
		if user.Username == "admin" {
			g.Response(http.StatusOK, ERROR_AUTH_ADMIN_CAN_NOT_DELETE, nil)
			return
		}
		if isExist {
			err := lkvs.DeleteUserInDB(user.Username)
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

// get summary of user and domain
func (lkvs *LKVS) apiGetSummary(c *gin.Context) {
	g := Gin{C: c}

	users := lkvs.GetAllUsers()
	domains, _ := lkvs.GetAllZones()

	type sum struct {
		Users   int `json:"users"`
		Domains int `json:"domains"`
	}

	s := &sum{
		Users:   len(users),
		Domains: len(domains),
	}

	g.Response(http.StatusOK, SUCCESS, s)
}

// accept slave's rsync request
func (lkvs *LKVS) rsync(c *gin.Context) {
	g := Gin{C: c}
	var err error

	slaveIP := GetClientIP(c)
	if !lkvs.slaveIsAllow(slaveIP) {
		g.Response(http.StatusForbidden, ERROR_SLAVE_IS_NOT_ALLOW, nil)
		return
	}

	type rDataStruct struct {
		Zones map[string]*Zone `json:"zones"`
		Users []*User          `json:"users"`
	}

	rData := &rDataStruct{}

	rData.Zones, err = lkvs.GetAllZones()
	if err != nil {
		g.Response(http.StatusOK, ERROR_GET_ZONES_FAIL, nil)
		return
	}
	rData.Users = lkvs.GetAllUsers()

	g.Response(http.StatusOK, SUCCESS, rData)

}

func (lkvs *LKVS) slaveIsAllow(ip string) (ok bool) {
	allow := 0
	for _, i := range lkvs.Slave {
		if i == ip {
			allow++
			break
		}
	}

	for _, i := range lkvs.Slave {
		if _ip := net.ParseIP(i); _ip == nil {
			_, _net, _ := net.ParseCIDR(i)
			if _net.Contains(net.ParseIP(ip)) {
				allow++
				break
			}
		}
	}

	if allow > 0 {
		ok = true
	} else {
		ok = false
	}
	return
}
