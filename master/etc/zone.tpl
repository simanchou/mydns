$TTL 600
@                           IN  SOA     {{.MainNS}}. admin.{{.Domain}}. ( {{.Serial}} 3H 15M 1W 1D ){{range .Nameserver}}
@                           IN  NS      {{.}}.{{end}}{{range .Records}}{{if eq .RecordType "A"}}
{{.Host}}                   {{.Ttl}}       IN  A      {{.PointsTo}}{{else if eq .RecordType "AAAA"}}
{{.Host}}                   {{.Ttl}}       IN  AAAA      {{.PointsTo}}{{else if eq .RecordType "CNAME"}}
{{.Host}}                   {{.Ttl}}       IN  CNAME      {{.PointsTo}}.{{else if eq .RecordType "MX"}}
{{.Host}}                   {{.Ttl}}       IN  MX        {{.MxPriority}}      {{.PointsTo}}.{{else if eq .RecordType "TXT"}}
{{.Host}}                   {{.Ttl}}       IN  TXT       "{{.PointsTo}}"{{else if eq .RecordType "CAA"}}
{{.CaaName}}                {{.Ttl}}       IN  CAA       {{.CaaFlags}}    {{.CaaTag}}    "{{.CaaValue}}"{{else if eq .RecordType "SRV"}}
{{.SrvService}}.{{.SrvProtocol}}.{{.SrvName}}.            {{.Ttl}}       IN  SRV  {{.SrvPriority}}    {{.SrvWeight}}    {{.SrvPort}}       {{.SrvTarget}}.{{end}}{{end}}
