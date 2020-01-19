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
	subDomain := c.Query("sub")
	rType := c.Query("type")
	ttl := com.StrTo(c.DefaultQuery("ttl", "600")).MustInt()

	valid := validation.Validation{}
	valid.Required(zoneName, "domain").Message("域名不能为空")
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(rType, "type").Message("记录类型不能为空")

	z := NewZone()
	if ! valid.HasErrors() {
		zoneName = strings.TrimSpace(zoneName)
		zoneName = strings.Trim(zoneName, ".")
		zoneName = zoneName + "."
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
			code, err := AddARecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "AAAA":
			code, err := AddAAAARecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "TXT":
			code, err := AddTXTRecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "CNAME":
			code, err := AddCNAMERecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "NS":
			code, err := AddNSRecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "MX":
			code, err := AddMXRecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "SRV":
			code, err := AddSRVRecordToZone(z, zoneName, rType, subDomain, ttl, c)
			if err != nil {
				g.Response(http.StatusOK, code, err)
				return
			}
		case "CAA":
			code, err := AddCAARecordToZone(z, zoneName, rType, subDomain, c)
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

}
