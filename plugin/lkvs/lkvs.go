package lkvs

import (
	"encoding/json"
	"errors"
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
	// BucketNameForDomain bucket name
	BucketNameForDomain = "domain"
	BucketNameForUser   = "user"
	defaultTTL          = 600
)

// LKVS local key-value storage
type LKVS struct {
	Next             plugin.Handler
	DB               *bolt.DB
	DBFile           string
	DBReadTimeout    int
	APIEngine        *gin.Engine
	APIPort          int
	TTL              uint32
	ZonesName        []string
	ZonesWithRecords map[string]Zone
	LastZoneUpdate   time.Time
}

// User user struct
type User struct {
	Username string `valid:"Required; MaxSize(50)" json:"username"`
	Password string `valid:"Required; MaxSize(50)" json:"password,omitempty"`
	CreateAt time.Time `json:"create_at"`
}

// Zone domain zone with records
type Zone struct {
	Zone    string             `json:"zone,omitempty"`
	User    string             `json:"user"`
	SOA     SOARecord          `json:"soa,omitempty"`
	Records map[string]*Record `json:"records,omitempty"`
}

// Record record of domain
type Record struct {
	ID         string `json:"id"`
	SubDomain  string `json:"subdomain"`
	TTL        uint32 `json:"ttl"`
	Type       string `json:"type"`
	IP         net.IP `json:"ip,omitempty"`
	Text       string `json:"text,omitempty"`
	Host       string `json:"host,omitempty"`
	Preference uint16 `json:"preference,omitempty"`
	Priority   uint16 `json:"priority,omitempty"`
	Weight     uint16 `json:"weight,omitempty"`
	Port       uint16 `json:"port,omitempty"`
	Target     string `json:"target,omitempty"`
	Flag       uint8  `json:"flag,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Value      string `json:"value,omitempty"`
}

// SOARecord type soa record
type SOARecord struct {
	TTL     uint32 `json:"ttl,omitempty"`
	Ns      string `json:"ns"`
	MBox    string `json:"mail"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTTL  uint32 `json:"minttl"`
}

/*
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
	MBox    string `json:"mail"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	MinTTL  uint32 `json:"minttl"`
}

*/

func NewUser(username, password string) *User {
	return &User{
		Username: username,
		Password: EncryptionPassword(password),
		CreateAt:time.Now(),
	}
}

func NewZone() *Zone {
	return &Zone{
		SOA: SOARecord{
			TTL:     defaultTTL,
			Refresh: 3600,
			Retry:   600,
			Expire:  86400,
			MinTTL:  3600,
		},
		Records: make(map[string]*Record),
	}
}

func NewRecord() *Record {
	return &Record{
		ID:  GenerateRecordID(),
		TTL: defaultTTL,
	}
}

func NewValidationError(errCode int, key, name, value string) *validation.Error {
	return &validation.Error{
		Message: GetCodeMsg(errCode),
		Key:     key,
		Name:    name,
		Value:   value,
	}
}

// AddARecordToZone add a record of type A
func AddARecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(host, "host").Message("主机IP不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "A"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.IP = net.ParseIP(host)

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.IP.String() == _record.IP.String() {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddAAAARecordToZone add a record of type AAAA
func AddAAAARecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(host, "host").Message("主机IP不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "AAAA"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.IP = net.ParseIP(host)

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.IP.String() == _record.IP.String() {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddTXTRecordToZone add a record of type TXT
func AddTXTRecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	text := c.Query("text")
	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(text, "text").Message("text不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "TXT"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.Text = DeleteSpace(text)

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.Text == _record.Text {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddCNAMERecordToZone add a record of type CNAME
func AddCNAMERecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	host := c.Query("host")
	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(host, "host").Message("目标主机不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "CNAME"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.Host = AddDotAtLast(host)

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.Host == _record.Host {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddMXRecordToZone add a record of type MX
func AddMXRecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	host := c.Query("host")
	preference := c.Query("preference")
	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(host, "host").Message("目标主机不能为空")
	valid.Required(preference, "preference").Message("优先级不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "MX"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.Host = AddDotAtLast(host)
		_r.Preference = uint16(com.StrTo(preference).MustInt())

		if _r.Preference == 0 {
			_r.Preference = 10
		}

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.Host == _record.Host {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}

	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddSRVRecordToZone add a record of type SRV
func AddSRVRecordToZone(z *Zone, ttl int, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	priority := c.Query("priority")
	weight := c.Query("weight")
	port := c.Query("port")
	target := c.Query("target")

	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(priority, "priority").Message("优先级不能为空")
	valid.Required(weight, "weight").Message("权重不能为空")
	valid.Required(port, "port").Message("服务端口不能为空")
	valid.Required(target, "target").Message("服务地址不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "SRV"
		_r.SubDomain = subDomain
		_r.TTL = CheckTTL(uint32(ttl))
		_r.Target = AddDotAtLast(target)
		_r.Priority = uint16(com.StrTo(priority).MustInt())
		_r.Weight = uint16(com.StrTo(weight).MustInt())
		_r.Port = uint16(com.StrTo(port).MustInt())

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.Target == _record.Target && _r.Port == _record.Port {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// AddCAARecordToZone add a record of type CAA
func AddCAARecordToZone(z *Zone, c *gin.Context) (errCode int, err *validation.Error) {
	subDomain := c.Query("sub")
	flag := c.Query("flag")
	tag := c.Query("tag")
	value := c.Query("value")

	valid := validation.Validation{}
	valid.Required(subDomain, "sub").Message("子域名不能为空")
	valid.Required(flag, "flag").Message("标志位不能为空")
	valid.Required(tag, "tag").Message("属性标签不能为空")
	valid.Required(value, "value").Message("属性标签的值不能为空")
	recordIsExist := false
	if !valid.HasErrors() {
		_r := NewRecord()
		_r.Type = "CAA"
		_r.SubDomain = subDomain
		_r.Flag = uint8(com.StrTo(flag).MustInt())
		_r.Tag = tag
		_r.Value = value

		for _, _record := range z.Records {
			if _r.SubDomain == _record.SubDomain && _r.Type == _record.Type && _r.Flag == _record.Flag && _r.Tag == _record.Tag && _r.Value == _record.Value {
				recordIsExist = true
				return ERROR_EXIST_RECORD, &validation.Error{
					Message: GetCodeMsg(ERROR_EXIST_RECORD),
					Key:     subDomain,
					Name:    subDomain,
					Value:   _r.IP.String(),
				}
			}
		}

		if !recordIsExist {
			z.Records[_r.ID] = _r
		}
	} else {
		for _, err := range valid.Errors {
			return INVALID_PARAMS, err
		}
	}
	return SUCCESS, nil
}

// EditARecord edit a record of type A
func EditARecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	host := DeleteSpace(c.Query("host"))
	ttl := DeleteSpace(c.Query("ttl"))

	if host == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "host + ttl", "host + ttl", "host,ttl不能全为空")
	}

	if host != "" {
		r.IP = net.ParseIP(host)
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditAAAARecord edit a record of type AAAA
func EditAAAARecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	host := DeleteSpace(c.Query("host"))
	ttl := DeleteSpace(c.Query("ttl"))

	if host == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "host + ttl", "host + ttl", "host,ttl不能全为空")
	}

	if host != "" {
		r.IP = net.ParseIP(host)
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditTXTRecord edit a record of type TXT
func EditTXTRecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	text := DeleteSpace(c.Query("text"))
	ttl := DeleteSpace(c.Query("ttl"))

	if text == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "text + ttl", "text + ttl", "text,ttl不能全为空")
	}

	if text != "" {
		r.Text = text
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditCNAMERecord edit a record of type CNAME
func EditCNAMERecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	host := DeleteSpace(c.Query("host"))
	ttl := DeleteSpace(c.Query("ttl"))

	if host == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "host + ttl", "host + ttl", "host,ttl不能全为空")
	}

	if host != "" {
		r.Host = AddDotAtLast(host)
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditMXRecord edit a record of type MX
func EditMXRecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	host := DeleteSpace(c.Query("host"))
	preference := DeleteSpace(c.Query("preference"))
	ttl := DeleteSpace(c.Query("ttl"))

	if host == "" && ttl == "" && preference == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "host + preference + ttl", "host + preference + ttl",
				"host,preference,ttl不能全为空")
	}

	if host != "" {
		r.Host = AddDotAtLast(host)
	}
	if preference != "" {
		r.Preference = uint16(com.StrTo(preference).MustInt())
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditSRVRecord edit a record of type SRV
func EditSRVRecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	target := DeleteSpace(c.Query("target"))
	port := DeleteSpace(c.Query("port"))
	priority := DeleteSpace(c.Query("priority"))
	weight := DeleteSpace(c.Query("weight"))
	ttl := DeleteSpace(c.Query("ttl"))

	if target == "" && port == "" && priority == "" && weight == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "target + port + priority + weight + ttl",
				"target + port + priority + weight + ttl",
				"target,port,priority,weight,ttl不能全为空")
	}

	if target != "" {
		r.Target = AddDotAtLast(target)
	}
	if port != "" {
		r.Port = uint16(com.StrTo(port).MustInt())
	}
	if priority != "" {
		r.Priority = uint16(com.StrTo(priority).MustInt())
	}
	if weight != "" {
		r.Weight = uint16(com.StrTo(weight).MustInt())
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

// EditCAARecord edit a record of type CAA
func EditCAARecord(z *Zone, r *Record, c *gin.Context) (errCode int, err *validation.Error) {
	flag := DeleteSpace(c.Query("flag"))
	tag := DeleteSpace(c.Query("tag"))
	value := DeleteSpace(c.Query("value"))
	ttl := DeleteSpace(c.Query("ttl"))

	if flag == "" && tag == "" && value == "" && ttl == "" {
		return INVALID_PARAMS,
			NewValidationError(INVALID_PARAMS, "flag + tag + value + ttl",
				"flag + tag + value + ttl",
				"flag,tag,value,ttl不能全为空")
	}

	if flag != "" {
		r.Flag = uint8(com.StrTo(flag).MustInt())
	}
	if tag != "" {
		r.Tag = tag
	}
	if value != "" {
		r.Value = value
	}
	if ttl != "" {
		r.TTL = uint32(com.StrTo(ttl).MustInt())
	}
	z.Records[r.ID] = r

	return SUCCESS, nil
}

func (lkvs *LKVS) serial() uint32 {
	return uint32(time.Now().Unix())
}

// LoadZones load all zones from db
func (lkvs *LKVS) LoadZones() {
	err := lkvs.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForDomain))
		c := b.Cursor()
		var _zoneName []string
		for k, v := c.First(); k != nil; k, v = c.Next() {
			_z := Zone{}
			err := json.Unmarshal(v, &_z)
			if err != nil {
				fmt.Println("decode fail, error: ", err)
				return err
			}
			_zoneName = append(_zoneName, _z.Zone)
			lkvs.ZonesName = _zoneName
			lkvs.ZonesWithRecords[_z.Zone] = _z
		}
		return nil
	})
	if err != nil {
		log.Println("load zones from db fail: ", err)
	}
}

/*
// SaveToDB save to db
func (lkvs *LKVS) SaveToDB(z *Zone) (err error) {
	err = lkvs.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForDomain))
		encode, err := json.Marshal(z)
		if err != nil {
			log.Println("encode fail, error: ", err)
			return err
		}
		return b.Put([]byte(z.Zone), encode)
	})
	return
}

*/

// Save save to db
func (lkvs *LKVS) Save(bn string, data interface{}) (err error) {
	err = lkvs.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bn))
		encode, err := json.Marshal(data)
		if err != nil {
			log.Println("encode fail, error: ", err)
			return err
		}
		switch data.(type) {
		case *Zone:
			z, _ := data.(*Zone)
			return b.Put([]byte(z.Zone), encode)
		case *User:
			u, _ := data.(*User)
			return b.Put([]byte(u.Username), encode)
		}
		return errors.New("unsupported storage type in db")
	})
	return
}

// DeleteZone delete zone in db
func (lkvs *LKVS) DeleteZoneInDB(zoneName string) (err error) {
	err = lkvs.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForDomain))
		err := b.Delete([]byte(zoneName))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

// A query of type A
func (lkvs *LKVS) A(name string, z Zone) (isCNAME bool, CNAMEHost string, answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)

	isWildcard := true
	isCNAME = false
	CNAMEHost = ""
	for _, _r := range z.Records {
		if _r.Type == "A" && _r.SubDomain == subDomain {
			isWildcard = false
		}
		if _r.Type == "CNAME" && _r.SubDomain == subDomain {
			isWildcard = false
			isCNAME = true
			CNAMEHost = _r.Host
		}
	}
	if isWildcard {
		subDomain = "*"
	}

	if isCNAME {
		for _, _r := range z.Records {
			if _r.Type == "CNAME" && _r.SubDomain == subDomain {
				r := new(dns.CNAME)
				r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
					Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
				r.Target = _r.Host
				answers = append(answers, r)
			} else {
				continue
			}
		}
	} else {
		for _, _r := range z.Records {
			if _r.Type == "A" && _r.SubDomain == subDomain {
				r := new(dns.A)
				r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
					Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
				r.A = _r.IP
				answers = append(answers, r)
			} else {
				continue
			}
		}
	}

	return
}

// A query of type AAAA
func (lkvs *LKVS) AAAA(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)

	isWildcard := true
	for _, _r := range z.Records {
		if _r.Type == "AAAA" && _r.SubDomain == subDomain {
			isWildcard = false
		}
	}
	if isWildcard {
		subDomain = "*"
	}

	for _, _r := range z.Records {
		if _r.Type == "AAAA" && _r.SubDomain == subDomain {
			r := new(dns.AAAA)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.AAAA = _r.IP
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

// A query of type TXT
func (lkvs *LKVS) TXT(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "TXT" && _r.SubDomain == subDomain {
			r := new(dns.TXT)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeTXT,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Txt = append(r.Txt, _r.Text)
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

// A query of type CNAME
func (lkvs *LKVS) CNAME(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "CNAME" && _r.SubDomain == subDomain {
			r := new(dns.CNAME)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Target = _r.Host
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

// A query of type MX
func (lkvs *LKVS) MX(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "MX" && _r.SubDomain == subDomain {
			r := new(dns.MX)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeMX,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Mx = _r.Host
			r.Preference = _r.Preference
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

// A query of type SRV
func (lkvs *LKVS) SRV(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "SRV" && _r.SubDomain == subDomain {
			r := new(dns.SRV)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeSRV,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Target = _r.Target
			r.Port = _r.Port
			r.Priority = _r.Priority
			r.Weight = _r.Weight
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

// A query of type CAA
func (lkvs *LKVS) CAA(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "CAA" && _r.SubDomain == subDomain {
			r := new(dns.CAA)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCAA,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Flag = _r.Flag
			r.Tag = _r.Tag
			r.Value = _r.Value
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

func (lkvs *LKVS) SOA(name string, z Zone) (answers, extras []dns.RR) {
	r := new(dns.SOA)
	if z.SOA.Ns == "" {
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeSOA,
			Class: dns.ClassINET, Ttl: defaultTTL}
		r.Ns = "ns1." + name
		r.Mbox = "hostmaster." + name
		r.Refresh = 86400
		r.Retry = 7200
		r.Expire = 3600
		r.Minttl = defaultTTL
	} else {
		r.Hdr = dns.RR_Header{Name: z.Zone, Rrtype: dns.TypeSOA,
			Class: dns.ClassINET, Ttl: CheckTTL(z.SOA.TTL)}
		r.Ns = z.SOA.Ns
		r.Mbox = z.SOA.MBox
		r.Refresh = z.SOA.Refresh
		r.Retry = z.SOA.Retry
		r.Expire = z.SOA.Expire
		r.Minttl = z.SOA.MinTTL
	}
	r.Serial = lkvs.serial()
	answers = append(answers, r)
	return
}

// A query of type NS
func (lkvs *LKVS) NS(name string, z Zone) (answers, extras []dns.RR) {
	subDomain := FindSubDomain(name, z.Zone)
	for _, _r := range z.Records {
		if _r.Type == "NS" && _r.SubDomain == subDomain {
			r := new(dns.NS)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
				Class: dns.ClassINET, Ttl: CheckTTL(_r.TTL)}
			r.Ns = _r.Host
			answers = append(answers, r)
		} else {
			continue
		}
	}
	return
}

func (lkvs *LKVS) UserIsExist(username string) (*User, bool) {
	u := User{}
	ok := true
	_ = lkvs.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForUser))
		v := b.Get([]byte(username))
		if v == nil {
			ok = false
		} else {
			json.Unmarshal(v, &u)
		}
		return nil
	})
	return &u, ok
}

func (lkvs *LKVS) CheckAuth(u *User) bool {
	err := lkvs.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForUser))
		v := b.Get([]byte(u.Username))

		var _u User
		err := json.Unmarshal(v, &_u)
		if err != nil {
			return err
		}
		if u.Password == _u.Password {
			return nil
		}
		return errors.New("wrong password")
	})
	if err != nil {
		return false
	}
	return true
}

// get all user from db
func (lkvs *LKVS) GetAllUsers() (users map[string]User) {
	users = make(map[string]User)
	_ = lkvs.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForUser))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			_u := User{}
			err := json.Unmarshal(v, &_u)
			if err != nil {
				return err
			}

			// hide password for user query
			_u.Password = ""
			users[_u.Username] = _u
		}
		return nil
	})
	return
}

// DeleteUser delete user in db
func (lkvs *LKVS) DeleteUserInDB(username string) (err error) {
	err = lkvs.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketNameForUser))
		err := b.Delete([]byte(username))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func CheckTTL(ttl uint32) uint32 {
	if ttl == 0 || ttl < defaultTTL {
		return defaultTTL
	}

	return ttl
}

func AddDotAtLast(str string) string {
	str = strings.TrimSpace(str)
	str = strings.Trim(str, ".")
	str = str + "."
	return str
}

func DeleteSpace(str string) string {
	_tmp := strings.Fields(str)
	if len(_tmp) == 0 {
		return ""
	}
	if len(_tmp) == 1 {
		return str
	}

	_str := ""
	for _, s := range _tmp {
		_str += s
	}

	return _str
}
