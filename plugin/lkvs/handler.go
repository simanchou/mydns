package lkvs

import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Handler interface
func (lkvs *LKVS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	qname := state.Name()
	qtype := state.Type()

	//lkvs.LoadZones()

	fmt.Println("qname in lkvs: ", qname)
	fmt.Println("qtype in lkvs: ", qtype)
	fmt.Println("zonsName in lkvs:", lkvs.ZonesName)

	zoneName := plugin.Zones(lkvs.ZonesName).Matches(qname)
	fmt.Println("zone in lkvs: ", zoneName)
	if zoneName == "" {
		fmt.Println("zone in lkvs is nil...")
		return plugin.NextOrFailure(qname, lkvs.Next, ctx, w, r)
	}


	zone := lkvs.ZonesWithRecords[zoneName]
	//subDomain := FindSubDomain(qname, zoneName)

	answers := make([]dns.RR, 0, 10)
	extras := make([]dns.RR, 0, 10)

	switch qtype {
	case "A":
		answers, extras = lkvs.A(qname, zone)
	case "AAAA":
		answers, extras = lkvs.AAAA(qname, zone)
	case "TXT":
		answers, extras = lkvs.TXT(qname, zone)
	case "CNAME":
		answers, extras = lkvs.CNAME(qname, zone)
	case "MX":
		answers, extras = lkvs.MX(qname, zone)
	case "SRV":
		answers, extras = lkvs.SRV(qname, zone)
	case "CAA":
		answers, extras = lkvs.CAA(qname, zone)
	case "SOA":
		answers, extras = lkvs.SOA(qname, zone)
	default:
		return lkvs.errorResponse(state, dns.RcodeNotImplemented, nil)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true
	m.Answer = append(m.Answer, answers...)
	m.Extra = append(m.Extra, extras...)

	state.SizeAndDo(m)
	m = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface
func (lkvs *LKVS) Name() string { return "lkvs" }

func (lkvs *LKVS) errorResponse(state request.Request, rcode int, err error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(state.Req, rcode)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true

	state.SizeAndDo(m)
	state.W.WriteMsg(m)
	return dns.RcodeSuccess, err
}
