package lkvs

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Handler interface
func (lkvs *LKVS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	logger.Debugf("request: %#v", r)
	logger.Debugf("state: %#v", state)
	logger.Debugf("state.writer: %#v", state.W)

	qname := state.Name()
	qtype := state.Type()

	//lkvs.LoadZones()
	zoneNames, err := lkvs.GetALLZoneName()
	if err != nil {
		return dns.RcodeBadName, err
	}
	logger.Debug("query name in lkvs: ", qname)
	logger.Debug("query type in lkvs: ", qtype)
	logger.Debug("zones name in lkvs:", zoneNames)

	zoneName := plugin.Zones(zoneNames).Matches(qname)
	logger.Debug("zone in lkvs: ", zoneName)
	if zoneName == "" {
		logger.Debug("zone in lkvs is nil...forward to next handler")
		return plugin.NextOrFailure(qname, lkvs.Next, ctx, w, r)
	}

	zones, err := lkvs.GetAllZones()
	if err != nil {
		return dns.RcodeBadName, err
	}

	zone := zones[zoneName]

	answers := make([]dns.RR, 0, 10)
	extras := make([]dns.RR, 0, 10)

	switch qtype {
	case "A":
		var (
			isCNAME   bool
			CNAMEHost string
		)
		isCNAME, CNAMEHost, answers, extras = lkvs.A(qname, zone)
		if isCNAME {
			zoneNameInCNAME := plugin.Zones(zoneNames).Matches(CNAMEHost)
			if zoneNameInCNAME == "" {
				answers1, extras1 := q(CNAMEHost, "", "A", "IN")
				answers = append(answers, answers1...)
				extras = append(extras, extras1...)
			} else {
				zoneInCNAME := zones[zoneNameInCNAME]
				_, _, answers2, extras2 := lkvs.A(CNAMEHost, zoneInCNAME)
				answers = append(answers, answers2...)
				extras = append(extras, extras2...)
			}
		}
	case "AAAA":
		answers, extras = lkvs.AAAA(qname, zone)
	case "TXT":
		answers, extras = lkvs.TXT(qname, zone)
	case "CNAME":
		answers, extras = lkvs.CNAME(qname, zone)
	case "MX":
		answers, extras = lkvs.MX(qname, zone)
	//case "SRV":
	//	answers, extras = lkvs.SRV(qname, zone)
	//case "CAA":
	//	answers, extras = lkvs.CAA(qname, zone)
	//case "SOA":
	//	answers, extras = lkvs.SOA(qname, zone)
	case "NS":
		answers, extras = lkvs.NS(qname, zone)
	default:
		return lkvs.errorResponse(state, dns.RcodeNotImplemented, nil)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, true, true
	m.Answer = append(m.Answer, answers...)
	m.Extra = append(m.Extra, extras...)
	logger.Debug("answers: %#v\n", answers)
	logger.Debug("m.answers: %#v\n", m.Answer)

	state.SizeAndDo(m)
	m = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

// Zone implements the Handler interface
func (lkvs *LKVS) Name() string { return "lkvs" }

func (lkvs *LKVS) errorResponse(state request.Request, rcode int, err error) (int, error) {
	m := new(dns.Msg)
	m.SetRcode(state.Req, rcode)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, false, true

	state.SizeAndDo(m)
	state.W.WriteMsg(m)
	return dns.RcodeSuccess, err
}
