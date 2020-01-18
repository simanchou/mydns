package lkvs

import (
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
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

	z := Zone{}
	if ! valid.HasErrors() {
		zoneName = strings.TrimSpace(zoneName)
		zoneName = strings.Trim(zoneName, ".")
		zoneName = zoneName + "."
		lkvs.LoadZones()

		if _z, ok := lkvs.ZonesWithRecords[zoneName];ok {
			z = _z
		}
		if z.Name == "" {
			z.Name = zoneName
		}

		switch strings.ToUpper(rType) {
		case "A":
			host := c.Query("host")
			var (
				_a ARecord
				_aRecord []ARecord
			)
			_a.TTL = uint32(ttl)
			_a.IP = net.ParseIP(host)

			if _, ok := z.Records.A[subDomain];ok {
				_index := len(z.Records.A[subDomain])
				for _, i := range z.Records.A[subDomain] {
					if i.IP.String() == host {
						g.Response(http.StatusOK, ERROR_EXIST_RECORD, nil)
						return
					}
				}
				_a.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
				z.Records.A[subDomain] = append(z.Records.A[subDomain], _a)
			} else {
				z.Records.A = make(map[string][]ARecord)
				_a.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
				_aRecord = append(_aRecord, _a)
				z.Records.A[subDomain] = _aRecord
			}
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
