package lkvs

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"log"
	"strconv"
	"time"
)

var RLKVS = &LKVS{ZonesWithRecords:make(map[string]Zone)}

func init() {
	var err error
	if RLKVS.DBFile == "" {
		RLKVS.DBFile = "dns.db"
	}
	if RLKVS.APIPort == 0 {
		RLKVS.APIPort = 5500
	}

	RLKVS.DB, err = bolt.Open(
		RLKVS.DBFile,
		0600,
		&bolt.Options{Timeout:time.Duration(RLKVS.DBReadTimout)*time.Second})
	log.Println("begin to open db file...")
	if err != nil {
		log.Fatalln("open db fail")
	}

	// init db for domain
	err = RLKVS.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketNameForDomain))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("init db for domain fail, error: %s", err)
	}
	// init db for user
	err = RLKVS.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(BucketNameForUser))
		if err != nil {
			return err
		}

		u := User{Username: "admin", Password: EncryptionPassword("123456")}
		encode, _ := json.Marshal(u)
		err = b.Put([]byte(u.Username), encode)
		return err
	})
	if err != nil {
		log.Fatalf("init db for user fail, error: %s", err)
	}

	RLKVS.TTL = 600
	RLKVS.LoadZones()
	RLKVS.InitRouter()

	caddy.RegisterPlugin("lkvs", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})

	fmt.Printf("%#v\n",RLKVS)
}

func setup(c *caddy.Controller) error {
	err := lkvsParse(c)
	if err != nil {
		return err
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		RLKVS.Next = next
		return RLKVS
	})
	return nil
}

func lkvsParse(c *caddy.Controller) (err error) {
	for c.Next() {
		if c.NextBlock() {
			for {
				switch c.Val() {
				case "lkvs_db_file":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.DBFile = c.Val()
				case "api_port":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.APIPort,err = strconv.Atoi(c.Val())
					if err !=nil {
						RLKVS.APIPort = 5500
					}
				case "timeout":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.DBReadTimout, err = strconv.Atoi(c.Val())
					if err != nil {
						RLKVS.DBReadTimout = 1
					}
				default:
					if c.Val() != "}" {
						return c.Errf("unknown property '%s'", c.Val())
					}
				}
				if !c.Next(){
					break
				}
			}
		}
		return
	}
	return
}