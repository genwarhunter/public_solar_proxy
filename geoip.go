package main

import (
	"github.com/oschwald/geoip2-golang"
	"io/ioutil"
	"log"
	"net"
)

/*
ссылка на скачивание базы
https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=1gszdqMxepVx&suffix=tar.gz
*/

var MaxMindDB *geoip2.Reader

func InitGeoIpDB() {
	var err error
	b, err := ioutil.ReadFile("GeoLite2-Country.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	MaxMindDB, err = geoip2.FromBytes(b)
	if err != nil {
		log.Fatal(err)
	}
}

func getCountryByIp(ip string) string {
	defer func() { recover() }()
	record, err := MaxMindDB.Country(net.ParseIP(ip))
	if err != nil {
		return ""
	}
	return record.Country.IsoCode
}
