package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Handler interface.
func (redis *Redis) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// fmt.Println("serveDNS")
	state := request.Request{W: w, Req: r}

	qname := state.Name()
	qtype := state.Type()

	fmt.Println("name : ", qname)
	fmt.Println("type : ", qtype)

	fmt.Printf("%#v\n", redis)
	fmt.Println(redis.LastZoneUpdate)
	if time.Since(redis.LastZoneUpdate) > zoneUpdateTime {
		redis.LoadZones()
	}

	zone := plugin.Zones(redis.Zones).Matches(qname)
	//zone := "example.com."
	fmt.Println("zone : ", zone)
	if zone == "" {
		fmt.Println("zone is blank...")
		return plugin.NextOrFailure(qname, redis.Next, ctx, w, r)
	}

	z := redis.load(zone)
	fmt.Println("zone from redis: ", z)
	if z == nil {
		return redis.errorResponse(state, zone, dns.RcodeServerFailure, nil)
	}

	location := redis.findLocation(qname, z)
	if len(location) == 0 { // empty, no results
		return redis.errorResponse(state, zone, dns.RcodeNameError, nil)
	}
	fmt.Println("location : ", location)

	answers := make([]dns.RR, 0, 10)
	extras := make([]dns.RR, 0, 10)

	record := redis.get(location, z)

	fmt.Printf("[Name]: %s [Type]: %s\n", qname, qtype)
	fmt.Printf("\t %+v\n", record)

	switch qtype {
	case "A":
		// Having a CNAME excludes A records, add cnames when querying for A records
		if len(record.CNAME) > 0 {
			//println("We have a cname in the record")
			answers2 := make([]dns.RR, 0, 10)
			extras2 := make([]dns.RR, 0, 10)
			answers2, extras2 = redis.CNAME(qname, z, record)
			answers = append(answers, answers2...)
			extras = append(extras, extras2...)
		} else {
			answers, extras = redis.A(qname, z, record)
		}
	case "AAAA":
		answers, extras = redis.AAAA(qname, z, record)
	case "CNAME":
		answers, extras = redis.CNAME(qname, z, record)
	case "TXT":
		answers, extras = redis.TXT(qname, z, record)
	case "NS":
		answers, extras = redis.NS(qname, z, record)
	case "MX":
		answers, extras = redis.MX(qname, z, record)
	case "SRV":
		answers, extras = redis.SRV(qname, z, record)
	case "SOA":
		answers, extras = redis.SOA(qname, z, record)
	case "CAA":
		answers, extras = redis.CAA(qname, z, record)
	default:
		return redis.errorResponse(state, zone, dns.RcodeNotImplemented, nil)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true

	m.Answer = append(m.Answer, answers...)
	m.Extra = append(m.Extra, extras...)

	// If there is a CNAME RR in the answers, solve the alias
	for _, CNAMERecord := range record.CNAME {
		fmt.Println("record.CNAME: ", CNAMERecord.Host)
		var query = strings.TrimSuffix(CNAMERecord.Host, "."+z.Name)
		records := redis.get(query, z)

		fmt.Printf("records from redis: \t %+v\n", records)
		answersN := make([]dns.RR, 0, 10)
		extrasN := make([]dns.RR, 0, 10)
		answersN, extrasN = redis.A(CNAMERecord.Host, z, records)
		m.Answer = append(m.Answer, answersN...)
		m.Extra = append(m.Extra, extrasN...)

		fmt.Println("query by trim in cname: ", query)
		fmt.Println("location from redis: ", location)
	}
	fmt.Println("-----END ANSWER")

	state.SizeAndDo(m)
	m = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (redis *Redis) Name() string { return "redis" }

func (redis *Redis) errorResponse(state request.Request, zone string, rcode int, err error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(state.Req, rcode)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true

	// m.Ns, _ = redis.SOA(state.Name(), zone, nil)

	state.SizeAndDo(m)
	state.W.WriteMsg(m)
	// Return success as the rcode to signal we have written to the client.
	return dns.RcodeSuccess, err
}
