package lkvs

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var RLKVS = &LKVS{}

func init() {
	caddy.RegisterPlugin("lkvs", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	err := lkvsParse(c)
	if err != nil {
		return err
	}

	if RLKVS.DBFile == "" {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalf("get current dir of db file fail, error: %s\n", err)
		}
		RLKVS.DBFile = path.Join(absDir, "dns.db")
	}
	if RLKVS.APIPort == 0 {
		RLKVS.APIPort = 5500
	}

	RLKVS.DB, err = bolt.Open(
		RLKVS.DBFile,
		0600,
		&bolt.Options{Timeout: time.Duration(RLKVS.DBReadTimeout) * time.Second})
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

		u := NewUser("admin", "123456")
		_, isExist := RLKVS.UserIsExist(u.Username)
		if !isExist {
			encode, _ := json.Marshal(u)
			err = b.Put([]byte(u.Username), encode)
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("init db for user fail, error: %s", err)
	}

	RLKVS.TTL = 600
	RLKVS.InitRouter()

	go RLKVS.APIStart()

	if RLKVS.Master != "" {
		go func() {
			for {
				log.Printf("begin to rsync from master %q\n", RLKVS.Master)
				sc, err := RLKVS.getRsync()
				if err != nil {
					log.Printf("rsync from master fail, error: %s\n", err)
				} else {
					log.Printf("rsync from master %q successful, zone total: %d\n", RLKVS.Master, sc)
				}
				log.Printf("next rsync after 60 seconds")
				time.Sleep(60 * time.Second)
			}
		}()
	}

	fmt.Printf("%#v\n", RLKVS)

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
				case "db_file":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.DBFile = c.Val()
				case "api_port":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.APIPort, err = strconv.Atoi(c.Val())
					if err != nil {
						RLKVS.APIPort = 5500
					}
				case "master":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.Master = c.Val()
				case "slave":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.Slave = strings.Split(c.Val(), ",")
				case "timeout":
					if !c.NextArg() {
						return c.ArgErr()
					}
					RLKVS.DBReadTimeout, err = strconv.Atoi(c.Val())
					if err != nil {
						RLKVS.DBReadTimeout = 1
					}
				default:
					if c.Val() != "}" {
						return c.Errf("unknown property '%s'", c.Val())
					}
				}
				if !c.Next() {
					break
				}
			}
		}
		return
	}
	return
}
