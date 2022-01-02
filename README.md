# mydns

my local dns base on bind with api.

- dns server power by [bind9](https://www.isc.org/bind/)
- api service power by [go-zero](https://github.com/zeromicro/go-zero)

## Getting started
> supported one master and multi salve
> 
> master api author info setup by ENV key API_KEY and API_SECRET
> 
> turn on recursion by ENV RECURSION=on ,default: off
> 
> turn off query log by ENV QUERY_LOG=off, default: on
>
> turn on dnssec by ENV DNSSEC=on ,default: off
>
> turn off dnssec if you test in LAN
- master 192.168.10.11
- 1st slave 192.168.10.12
- 2nd slave 192.168.10.13
- 3rd slave 192.168.10.14

### start master on host 192.168.10.11
```
sudo docker run --rm --name dns \
    -e API_KEY=123456 \
    -e API_SECRET=abcdef \
    -e SLAVES=192.168.10.12,192.168.10.13,192.168.10.14  \
    -e RECURSION=on \
    -e QUERY_LOG=off \
    -p53:53/udp \
    -p53:53 \
    -p52099:52099 \
    docker.io/simanchou/mydns-master
```

### start slave on host 192.168.10.12, 192.168.10.13, 192.168.10.14
```
sudo docker run --rm --name dns \
    -e MASTER_IP=192.168.10.11 \
    -e RECURSION=on \
    -e QUERY_LOG=off \
    -p53:53/udp \
    -p53:53 \
    -p52099:52099 \
    docker.io/simanchou/mydns-slave
```

### add a domain by the master's api
> turn on SSL strict, self-signed by openssl before app startup
```
curl --location --request POST 'https://192.168.18.11:52099/domain' \
--insecure \
--header 'Authorization: sso-key 123456:abcdef' \
--header 'Content-Type: application/json' \
--data-raw '{
    "domain": "example.com",
    "serial": 1,
    "records": [
        {
            "record_type": "NS",
            "host": "@",
            "points_to": "ns1.mydns.local",
            "ttl": 86400
        },
        {
            "record_type": "NS",
            "host": "@",
            "points_to": "ns2.mydns.local",
            "ttl": 86400
        },
        {
            "record_type": "A",
            "host": "@",
            "points_to": "1.1.1.1",
            "ttl": 600
        },
        {
            "record_type": "A",
            "host": "a1",
            "points_to": "2.2.2.2",
            "ttl": 600
        },
        {
            "record_type": "A",
            "host": "*",
            "points_to": "3.3.3.3",
            "ttl": 600
        },
        {
            "record_type": "CNAME",
            "host": "c1",
            "points_to": "a1.example.com",
            "ttl": 600
        },
        {
            "record_type": "MX",
            "host": "@",
            "points_to": "a2.example.com",
            "ttl": 600,
            "mx_priority": 10
        },
        {
            "record_type": "TXT",
            "host": "t1",
            "points_to": "this is a txt record",
            "ttl": 600
        },
        {
            "record_type": "CAA",
            "ttl": 86400,
            "caa_name": "@",
            "caa_flags": "0",
            "caa_tag": "issue",
            "caa_value": "ca.abc.com"
        },
        {
            "record_type": "SRV",
            "ttl": 86400,
            "srv_service": "_mysvc",
            "srv_protocol": "_tcp",
            "srv_name": "example.com",
            "srv_target": "a1.example.com",
            "srv_priority": 0,
            "srv_weight": 10,
            "srv_port": 3000
        }
    ]
}'
```
### valid dns record by dig
```
$ dig SOA example.com @192.168.10.11 +short
ns1.mydns.local. admin.example.com. 1 10800 900 604800 86400
$ dig NS example.com @192.168.10.11 +short
ns2.mydns.local.
ns1.mydns.local.
$ dig A example.com @192.168.10.11 +short
1.1.1.1
$ dig A a1.example.com @192.168.10.11 +short
2.2.2.2
$ dig A wildcard-record-1.example.com @192.168.10.11 +short
3.3.3.3
$ dig A wildcard-record-2.example.com @192.168.10.11 +short
3.3.3.3
$ dig MX example.com @192.168.10.11 +short
10 a2.example.com.
$ dig TXT t1.example.com @192.168.10.11 +short
"this is a txt record"
$ dig CAA example.com @192.168.10.11 +short
0 issue "ca.abc.com"
$ dig SRV _mysvc._tcp.example.com @192.168.10.11 +short
0 10 3000 a1.example.com.

```
### get more detail for api by swagger use master/mydns.json
```
cd master;sudo docker run -d --name doc-mydns -p 7000:8080 -e SWAGGER_JSON=/mydns/mydns.json -v $PWD:/mydns swaggerapi/swagger-ui
```
> browse http://127.0.0.1:7000/ to get more detail of api after the command above
