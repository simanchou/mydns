#!/bin/sh

set -e

# logging functions
LOG() {
    local type="$1"; shift
    # accept argument string or stdin
    local text="$*"; if [ "$#" -eq 0 ]; then text="$(cat)"; fi
    local dt; dt="$(date +"%Y-%m-%d %H:%M:%S")"
    printf '%s [%b] : %s\n' "$dt" "$type"  "$text"
}
LOG_INFO() {
    Info="\e[42;37mInfo\e[0m"
    LOG "$Info" "$@"
}
LOG_WARN() {
    Warn="\e[43;37mWarn\e[0m"
    LOG "$Warn" "$@" >&2
}
LOG_ERROR() {
    ERROR="\e[41;37mERROR\e[0m"
    LOG "$ERROR" "$@" >&2
    exit 1
}

LOG_INFO "begin to set supervisor run in non-daemon mode"
sed -i 's/^\(\[supervisord\]\)$/\1\nnodaemon=true/' /etc/supervisord.conf
LOG_INFO "supervisor setup to non-daemon mode"

LOG_INFO "begin to init named conf"
LOG_INFO "found slave ip by ENV key \"SLAVES\": ${SLAVES}"
IFS=,
SLAVE_LIST=""
for ip in $SLAVES;do
    SLAVE_LIST="${ip};${SLAVE_LIST}"
done
mv /etc/bind/named.conf.options.master /etc/bind/named.conf.options
sed -i "s/SLAVE_LIST/$SLAVE_LIST/" /etc/bind/named.conf.options
if [ "$RECURSION" = "on" ];then
    RECURSION_ENABLE=yes
  else
    RECURSION_ENABLE=no
fi
sed -i "s/RECURSION_ENABLE/$RECURSION_ENABLE/" /etc/bind/named.conf.options
if [ "$DNSSEC" = "on" ];then
    DNSSEC_ENABLE=yes
  else
    DNSSEC_ENABLE=no
fi
sed -i "s/DNSSEC_ENABLE/$DNSSEC_ENABLE/" /etc/bind/named.conf.options
if [ "$QUERY_LOG" = "off" ];then
    sed -i 's/.*named.conf.logging.*//g' /etc/bind/named.conf
fi
cat /etc/bind/named.conf.options
LOG_INFO "init named conf successful"

chown -R named:named /etc/bind
chown -R named:named /var/cache/bind

# ssl
SSL_PATH=/app/ssl
PASSWORD=$(openssl rand -hex 8)
if [ -s ${SSL_PATH}/ssl.crt ] || [ -s ${SSL_PATH}/cert.pem ] || [ -s ${SSL_PATH}/key.pem ] || [ -n "${SKIP_SSL_GENERATE}" ]; then
    LOG_INFO "Skipping SSL certificate generation"
else
    LOG_INFO "Generating self-signed certificate"

    mkdir -p ${SSL_PATH}

    # Generating signing SSL private key
    openssl genrsa -des3 -passout pass:${PASSWORD} -out ${SSL_PATH}/key.pem 2048

    # Removing passphrase from private key
    cp ${SSL_PATH}/key.pem ${SSL_PATH}/key.pem.orig
    openssl rsa -passin pass:${PASSWORD} -in ${SSL_PATH}/key.pem.orig -out ${SSL_PATH}/key.pem

    # Generating certificate signing request
    openssl req -new -key ${SSL_PATH}/key.pem -out ${SSL_PATH}/cert.csr -subj "/C=CN/ST=Beijing/L=Beijing/O=Example Co., Ltd./OU=IT/CN=Example"

    # Generating self-signed certificate
    openssl x509 -req -days 3650 -in ${SSL_PATH}/cert.csr -signkey ${SSL_PATH}/key.pem -out ${SSL_PATH}/cert.pem

    LOG_INFO "Self-signed certificate generate done"
fi

exec "$@"
