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
	api := engine.Group("/api")
	{
		api.GET("/domain", lkvs.apiGetZones)
		api.POST("/domain", lkvs.apiAddZone)
		api.DELETE("/domain", lkvs.apiDeleteZone)

		api.GET("/record", lkvs.apiGetRecords)
		api.POST("/record", lkvs.apiAddRecord)
		api.PUT("/record", lkvs.apiEditRecord)
		api.DELETE("/record", lkvs.apiDeleteRecord)
	}

	lkvs.APIEngine = engine
}

// register
func (lkvs *LKVS) register(c *gin.Context) {
	g := Gin{C:c}
	username := DeleteSpace(c.Query("username"))
	password := DeleteSpace(c.Query("password"))

	valid := validation.Validation{}
	u := User{Username: username, Password: password}
	ok, _ := valid.Valid(&u)

	if ok {
		isExist := lkvs.UserIsExist(u.Username)
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

// get all zones
func (lkvs *LKVS) apiGetZones(c *gin.Context) {
	g := Gin{C: c}
	lkvs.LoadZones()
	data := lkvs.ZonesWithRecords
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
			z.Name = zoneName
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
		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			if _z.Records == nil {
				delete(lkvs.ZonesWithRecords, zoneName)
			} else {
				g.Response(http.StatusOK, ERROR_CAN_NOT_DELETE_ZONE_WHEN_RECORD_NOT_NIL, nil)
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

		if data, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			g.Response(http.StatusOK, SUCCESS, data)
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

		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			var z *Zone
			z = &_z
			if z.Records == nil {
				z.Records = make(map[string]*Record)
			}
			fmt.Println("rType: ", rType)
			fmt.Printf("zone: %#v\n", z)
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

		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			var z *Zone
			z = &_z

			fmt.Println("id: ", id)
			fmt.Printf("zone: %#v\n", z)

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
		if _z, ok := lkvs.ZonesWithRecords[zoneName]; ok {
			z = &_z
			if _, ok := z.Records[id]; ok {
				delete(z.Records, id)
			} else {
				g.Response(http.StatusOK, ERROR_NOT_EXIST_RECORD, nil)
				return
			}
		} else {
			g.Response(http.StatusOK, ERROR_NOT_EXIST_ZONE, nil)
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
