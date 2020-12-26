package lkvs

import (
	"github.com/boltdb/bolt"
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
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
			logger.Fatalf("get current dir of db file fail, error: %s", err.Error())
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
	logger.Debug("begin to open db file...")
	if err != nil {
		logger.Fatalln("open db fail")
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
		logger.Fatalf("init db for domain fail, error: %s", err.Error())
	}
	// init db for user
	err = RLKVS.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketNameForUser))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Fatalf("init db for user fail, error: %s", err.Error())
	}
	// init admin user
	u := NewUser("admin", "123456")
	u.Avatar = "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif"
	u.Nickname = "super administrator"
	u.Roles = append(u.Roles, "admin")
	_, isExist := RLKVS.UserIsExistByName(u.Username)
	if !isExist {
		err = RLKVS.Save(BucketNameForUser, u)
		if err != nil {
			logger.Fatalf("init admin user fail, error: %s", err.Error())
		}
	}

	RLKVS.TTL = 600
	RLKVS.InitRouter()

	go RLKVS.APIStart()

	if RLKVS.Master != "" {
		go func() {
			for {
				logger.Debugf("begin to rsync from master %q", RLKVS.Master)
				sc, err := RLKVS.getRsync()
				if err != nil {
					logger.Errorf("rsync from master fail, error: %s", err.Error())
				} else {
					logger.Debugf("rsync from master %q successful, zone total: %d", RLKVS.Master, sc)
				}
				logger.Debugf("next rsync after 60 seconds")
				time.Sleep(60 * time.Second)
			}
		}()
	}

	logger.Debugf("lkvs init successful, %#v", RLKVS)

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
