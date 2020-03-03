package lkvs

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func q(queryName, nameserver, queryType, queryClass string)(answers, extras []dns.RR) {
	var (
		qtype  []uint16
		qclass []uint16
		qname  []string
	)

	qname = append(qname, queryName)

	if len(queryType) == 0 {
		qtype = append(qtype, dns.TypeA)
	} else {
		if k, ok := dns.StringToType[strings.ToUpper(queryType)];ok {
			qtype = append(qtype, k)
		} else {
			qtype = append(qtype, dns.TypeNULL)
		}
	}
	if len(queryClass) == 0 {
		qclass = append(qclass, dns.ClassINET)
	} else {
		if k, ok := dns.StringToClass[strings.ToUpper(queryClass)];ok{
			qclass = append(qclass, k)
		} else {
			qclass = append(qclass, dns.ClassNONE)
		}
	}

	if len(nameserver) == 0 {
		conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		nameserver = "@" + conf.Servers[0]
	}

	nameserver = string([]byte(nameserver)[1:]) // chop off @
	// if the nameserver is from /etc/resolv.conf the [ and ] are already
	// added, thereby breaking net.ParseIP. Check for this and don't
	// fully qualify such a name
	if nameserver[0] == '[' && nameserver[len(nameserver)-1] == ']' {
		nameserver = nameserver[1 : len(nameserver)-1]
	}
	if i := net.ParseIP(nameserver); i != nil {
		nameserver = net.JoinHostPort(nameserver, strconv.Itoa(53))
	} else {
		nameserver = dns.Fqdn(nameserver) + ":" + strconv.Itoa(53)
	}
	c := new(dns.Client)
	c.Net = "udp"
	c.DialTimeout = 2*time.Second
	c.ReadTimeout = 2*time.Second
	c.WriteTimeout = 2*time.Second

	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Opcode:            dns.OpcodeQuery,
		},
		Question: make([]dns.Question, 1),
	}
	m.Rcode = dns.RcodeSuccess

	qt := dns.TypeA
	qc := uint16(dns.ClassINET)

	for i, v := range qname {
		if i < len(qtype) {
			qt = qtype[i]
		}
		if i < len(qclass) {
			qc = qclass[i]
		}
		m.Question[0] = dns.Question{Name: dns.Fqdn(v), Qtype: qt, Qclass: qc}
		m.Id = dns.Id()

		r, _, err := c.Exchange(m, nameserver)

		switch err {
		case nil:
			answers = r.Answer
			extras = r.Extra
		default:
			fmt.Printf(";; %s\n", err.Error())
			continue
		}
		if r.Id != m.Id {
			fmt.Fprintf(os.Stderr, "Id mismatch\n")
			return
		}

	}
	return
}