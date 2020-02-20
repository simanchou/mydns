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
	api := engine.Group("/api")
	{
		api.GET("/domain", lkvs.apiGetZones)
		api.POST("/domain", lkvs.apiAddZone)
		api.PUT("/domain", lkvs.apiEditZone)
		api.DELETE("/domain", lkvs.apiDeleteZone)

		api.DELETE("/record",lkvs.apiDeleteRecord)
	}

	lkvs.APIEngine = engine
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
	rType := c.Query("type")
	ttl := com.StrTo(c.DefaultQuery("ttl", "600")).MustInt()

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(rType, "type").Message("记录类型不能为空")

	z := NewZone()
	if ! valid.HasErrors() {
		zoneName = strings.TrimSpace(zoneName)
		zoneName = strings.Trim(zoneName, ".")
		zoneName = zoneName + "."
		z.SOA.MBox = fmt.Sprintf("admin.%s",zoneName)
		z.SOA.Ns = "ns.mydns.local."

		lkvs.LoadZones()

		if _z, ok := lkvs.ZonesWithRecords[zoneName];ok {
			z = &_z
		}
		if z.Name == "" {
			z.Name = zoneName
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
		}
	} else {
		for _, err := range valid.Errors {
			g.Response(http.StatusOK, INVALID_PARAMS, err)
			return
		}
	}
	err := lkvs.SaveToDB(z)
	if err != nil {
		g.Response(http.StatusInternalServerError, ERROR_ADD_ZONE_FAIL, nil)
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}

// edit zone
func (lkvs *LKVS) apiEditZone(c *gin.Context) {

}

// delete zone
func (lkvs *LKVS) apiDeleteZone(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")

	if ! valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		if _z,ok := lkvs.ZonesWithRecords[zoneName];ok{
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

// delete record
func (lkvs *LKVS) apiDeleteRecord(c *gin.Context) {
	g := Gin{C: c}
	zoneName := c.Query("domain")
	id := c.Query("id")

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(id, "id").Message("记录ID不能为空")

	var z *Zone
	if ! valid.HasErrors() {
		zoneName = AddDotAtLast(zoneName)
		if _z,ok := lkvs.ZonesWithRecords[zoneName];ok{
			z = &_z
			if _, ok := z.Records[id];ok{
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

	err := lkvs.SaveToDB(z)
	if err != nil {
		g.Response(http.StatusInternalServerError, ERROR_DELETE_RECORD_FAIL, nil)
	}
	g.Response(http.StatusOK, SUCCESS, nil)
}
