package lkvs

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
	"log"
	"net"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const (
	// BucketName bucket name
	BucketName = "domain"
	defaultTTL = 600
)

// LKVS local key-value storage
type LKVS struct {
	Next             plugin.Handler
	DB               *bolt.DB
	DBFile           string
	DBReadTimout     int
	APIEngine        *gin.Engine
	APIPort          int
	TTL              uint32
	ZonesName        []string
	ZonesWithRecords map[string]Zone
	LastZoneUpdate   time.Time
}

// Zone domain zone with records
type Zone struct {
	Name    string `json:"name,omitempty"`
	Records Record `json:"records,omitempty"`
}

// Record record of domain
type Record struct {
	A     map[string][]ARecord     `json:"a,omitempty"`
	AAAA  map[string][]AAAARecord  `json:"aaaa,omitempty"`
	TXT   map[string][]TXTRecord   `json:"txt,omitempty"`
	CNAME map[string][]CNAMERecord `json:"cname,omitempty"`
	NS    map[string][]NSRecord    `json:"ns,omitempty"`
	MX    map[string][]MXRecord    `json:"mx,omitempty"`
	SRV   map[string][]SRVRecord   `json:"srv,omitempty"`
	CAA   map[string][]CAARecord   `json:"caa,omitempty"`
	SOA   SOARecord                `json:"soa,omitempty"`
}

// ARecord type a record
type ARecord struct {
	ID  string `json:"id"`
	TTL uint32 `json:"ttl,omitempty"`
	IP  net.IP `json:"ip"`
}

// AAAARecord type aaaa record
type AAAARecord struct {
	ID  string `json:"id"`
	TTL uint32 `json:"ttl,omitempty"`
	IP  net.IP `json:"ip"`
}

// TXTRecord type txt record
type TXTRecord struct {
	ID   string `json:"id"`
	TTL  uint32 `json:"ttl,omitempty"`
	Text string `json:"text"`
}

// CNAMERecord type cname record
type CNAMERecord struct {
	ID   string `json:"id"`
	TTL  uint32 `json:"ttl,omitempty"`
	Host string `json:"host"`
}

// NSRecord type ns record
type NSRecord struct {
	ID   string `json:"id"`
	TTL  uint32 `json:"ttl,omitempty"`
	Host string `json:"host"`
}

// MXRecord type mx record
type MXRecord struct {
	ID         string `json:"id"`
	TTL        uint32 `json:"ttl,omitempty"`
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

// SRVRecord type srv record
type SRVRecord struct {
	ID       string `json:"id"`
	TTL      uint32 `json:"ttl,omitempty"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
	Target   string `json:"target"`
}

// CAARecord type caa record
type CAARecord struct {
	ID    string `json:"id"`
	Flag  uint8  `json:"flag"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// SOARecord type soa record
type SOARecord struct {
	TTL     uint32 `json:"ttl,omitempty"`
	Ns      string `json:"ns"`
	MBox    string `json:"MBox"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTTL  uint32 `json:"minttl"`
}

func NewZone() *Zone {
	return &Zone{
		Records:Record{
			A:     make(map[string][]ARecord),
			AAAA:  make(map[string][]AAAARecord),
			TXT:   make(map[string][]TXTRecord),
			CNAME: make(map[string][]CNAMERecord),
			NS:    make(map[string][]NSRecord),
			MX:    make(map[string][]MXRecord),
			SRV:   make(map[string][]SRVRecord),
			CAA:   make(map[string][]CAARecord),
			SOA:   SOARecord{},
		},
	}
}

// AddARecordToZone add a record of type A
func AddARecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(host, "host").Message("主机IP不能为空")
	if ! valid.HasErrors(){
		var (
			_a ARecord
			_aRecord []ARecord
		)
		_a.TTL = CheckTTL(uint32(ttl))
		_a.IP = net.ParseIP(host)

		if _, ok := z.Records.A[subDomain];ok {
			_index := len(z.Records.A[subDomain])
			for _, i := range z.Records.A[subDomain] {
				if i.IP.String() == host {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.IP.String(),
					}
				}
			}
			_a.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
			z.Records.A[subDomain] = append(z.Records.A[subDomain], _a)
		} else {
			if z.Records.A == nil {
				z.Records.A = make(map[string][]ARecord)
			}
			_a.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
			_aRecord = append(_aRecord, _a)
			z.Records.A[subDomain] = _aRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddAAAARecordToZone add a record of type AAAA
func AddAAAARecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(host, "host").Message("主机IP不能为空")
	if ! valid.HasErrors(){
		var (
			_aaaa       AAAARecord
			_aaaaRecord []AAAARecord
		)
		_aaaa.TTL = CheckTTL(uint32(ttl))
		_aaaa.IP = net.ParseIP(host)

		if _, ok := z.Records.AAAA[subDomain];ok {
			_index := len(z.Records.AAAA[subDomain])
			for _, i := range z.Records.AAAA[subDomain] {
				if i.IP.String() == host {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.IP.String(),
					}
				}
			}
			_aaaa.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
			z.Records.AAAA[subDomain] = append(z.Records.AAAA[subDomain], _aaaa)
		} else {
			if z.Records.AAAA == nil {
				z.Records.AAAA = make(map[string][]AAAARecord)
			}
			_aaaa.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
			_aaaaRecord = append(_aaaaRecord, _aaaa)
			z.Records.AAAA[subDomain] = _aaaaRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}


// AddTXTRecordToZone add a record of type TXT
func AddTXTRecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	text := c.Query("text")
	valid := validation.Validation{}
	valid.Required(text, "text").Message("主机IP不能为空")
	if ! valid.HasErrors(){
		var (
			_txt       TXTRecord
			_txtRecord []TXTRecord
		)
		_txt.TTL = CheckTTL(uint32(ttl))
		_txt.Text = text

		if _, ok := z.Records.TXT[subDomain];ok {
			_index := len(z.Records.TXT[subDomain])
			for _, i := range z.Records.TXT[subDomain] {
				if i.Text == text {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.Text,
					}
				}
			}
			_txt.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+text+"|"+fmt.Sprintf("%d",_index))
			z.Records.TXT[subDomain] = append(z.Records.TXT[subDomain], _txt)
		} else {
			if z.Records.TXT == nil {
				z.Records.TXT = make(map[string][]TXTRecord)
			}
			_txt.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+text+"|"+fmt.Sprintf("%d",0))
			_txtRecord = append(_txtRecord, _txt)
			z.Records.TXT[subDomain] = _txtRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddCNAMERecordToZone add a record of type CNAME
func AddCNAMERecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(host, "host").Message("目标主机不能为空")
	if ! valid.HasErrors(){
		var (
			_cname       CNAMERecord
			_cnameRecord []CNAMERecord
		)

		host = strings.TrimSpace(host)
		host = strings.Trim(host, ".")
		host = host + "."

		_cname.TTL = CheckTTL(uint32(ttl))
		_cname.Host = host

		if _, ok := z.Records.CNAME[subDomain];ok {
			_index := len(z.Records.CNAME[subDomain])
			for _, i := range z.Records.CNAME[subDomain] {
				if i.Host == host {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.Host,
					}
				}
			}
			_cname.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
			z.Records.CNAME[subDomain] = append(z.Records.CNAME[subDomain], _cname)
		} else {
			if z.Records.CNAME == nil {
				z.Records.CNAME = make(map[string][]CNAMERecord)
			}
			_cname.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
			_cnameRecord = append(_cnameRecord, _cname)
			z.Records.CNAME[subDomain] = _cnameRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddNSRecordToZone add a record of type NS
func AddNSRecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(host, "host").Message("目标主机不能为空")
	if ! valid.HasErrors(){
		var (
			_ns       NSRecord
			_nsRecord []NSRecord
		)

		host = strings.TrimSpace(host)
		host = strings.Trim(host, ".")
		host = host + "."

		_ns.TTL = CheckTTL(uint32(ttl))
		_ns.Host = host

		if _, ok := z.Records.NS[subDomain];ok {
			_index := len(z.Records.NS[subDomain])
			for _, i := range z.Records.NS[subDomain] {
				if i.Host == host {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.Host,
					}
				}
			}
			_ns.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
			z.Records.NS[subDomain] = append(z.Records.NS[subDomain], _ns)
		} else {
			if z.Records.NS == nil {
				z.Records.NS = make(map[string][]NSRecord)
			}
			_ns.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
			_nsRecord = append(_nsRecord, _ns)
			z.Records.NS[subDomain] = _nsRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddMXRecordToZone add a record of type MX
func AddMXRecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	host := c.Query("host")
	preference := c.Query("preference")
	valid := validation.Validation{}
	valid.Required(host, "host").Message("目标主机不能为空")
	valid.Required(preference, "preference").Message("优先级不能为空")
	if ! valid.HasErrors(){
		var (
			_mx       MXRecord
			_mxRecord []MXRecord
		)

		host = strings.TrimSpace(host)
		host = strings.Trim(host, ".")
		host = host + "."

		_mx.TTL = CheckTTL(uint32(ttl))
		_mx.Host = host
		_mx.Preference = uint16(com.StrTo(preference).MustInt())

		if _mx.Preference == 0 {
			_mx.Preference = 10
		}

		if _, ok := z.Records.MX[subDomain];ok {
			_index := len(z.Records.MX[subDomain])
			for _, i := range z.Records.MX[subDomain] {
				if i.Host == host {
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.Host,
					}
				}
			}
			_mx.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",_index))
			z.Records.MX[subDomain] = append(z.Records.MX[subDomain], _mx)
		} else {
			if z.Records.MX == nil {
				z.Records.MX = make(map[string][]MXRecord)
			}
			_mx.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+host+"|"+fmt.Sprintf("%d",0))
			_mxRecord = append(_mxRecord, _mx)
			z.Records.MX[subDomain] = _mxRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddSRVRecordToZone add a record of type SRV
func AddSRVRecordToZone(z *Zone, zoneName, rType, subDomain string,  ttl int, c *gin.Context) (errCode int, err *validation.Error){
	priority := c.Query("priority")
	weight := c.Query("weight")
	port := c.Query("port")
	target := c.Query("target")

	valid := validation.Validation{}
	valid.Required(priority, "priority").Message("优先级不能为空")
	valid.Required(weight, "weight").Message("权重不能为空")
	valid.Required(port, "port").Message("服务端口不能为空")
	valid.Required(target, "target").Message("服务地址不能为空")
	if ! valid.HasErrors(){
		var (
			_srv       SRVRecord
			_srvRecord []SRVRecord
		)

		target = strings.TrimSpace(target)
		target = strings.Trim(target, ".")
		target = target + "."

		_srv.TTL = CheckTTL(uint32(ttl))
		_srv.Priority = uint16(com.StrTo(priority).MustInt())
		_srv.Weight = uint16(com.StrTo(weight).MustInt())
		_srv.Port = uint16(com.StrTo(port).MustInt())
		_srv.Target = target

		if _, ok := z.Records.SRV[subDomain];ok {
			_index := len(z.Records.SRV[subDomain])
			for _, i := range z.Records.SRV[subDomain] {
				if i.Target == _srv.Target && i.Port == _srv.Port{
					return ERROR_EXIST_RECORD, &validation.Error{
						Message:GetCodeMsg(ERROR_EXIST_RECORD),
						Key:subDomain,
						Name:subDomain,
						Value:i.Target,
					}
				}
			}
			_srv.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+_srv.Target+"|"+fmt.Sprintf("%d",_srv.Port)+"|"+fmt.Sprintf("%d",_index))
			z.Records.SRV[subDomain] = append(z.Records.SRV[subDomain], _srv)
		} else {
			if z.Records.SRV == nil {
				z.Records.SRV = make(map[string][]SRVRecord)
			}
			_srv.ID = GenerateRecordID(zoneName+"|"+rType+"|"+subDomain+"|"+_srv.Target+"|"+fmt.Sprintf("%d",_srv.Port)+"|"+fmt.Sprintf("%d",0))
			_srvRecord = append(_srvRecord, _srv)
			z.Records.SRV[subDomain] = _srvRecord
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// LoadZones load all zones from db
func (lkvs *LKVS) LoadZones() {
	err := lkvs.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			_z := Zone{}
			err := json.Unmarshal(v, &_z)
			if err != nil {
				fmt.Println("decode fail, error: ", err)
				return err
			}
			lkvs.ZonesName = append(lkvs.ZonesName, _z.Name)
			lkvs.ZonesWithRecords[_z.Name] = _z
		}
		return nil
	})
	if err != nil {
		log.Println("load zones from db fail: ", err)
	}
}

// SaveToDB save to db
func (lkvs *LKVS) SaveToDB(z *Zone) (err error) {
	err = lkvs.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketName))
		encode, err := json.Marshal(z)
		if err != nil {
			log.Println("encode fail, error: ", err)
			return err
		}
		return b.Put([]byte(z.Name), encode)
	})
	return
}

// A query type a record
func (lkvs *LKVS) A(name string, z Zone, record Record) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Name)
	for _, a := range record.A[subDomain] {
		if a.IP == nil {
			continue
		}
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
			Class: dns.ClassINET, Ttl: CheckTTL(a.TTL)}
		r.A = a.IP
		answers = append(answers, r)
	}
	return
}

func CheckTTL(ttl uint32) uint32 {
	if ttl == 0 || ttl < defaultTTL{
		return defaultTTL
	}

	return ttl
}
