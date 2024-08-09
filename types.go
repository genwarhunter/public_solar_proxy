package main

import (
	"time"
)

type proxy struct {
	ip   uint32
	port uint16
}

type proxyInfo struct {
	login    string
	password string
	protocol byte
}

type proxyForGraph struct {
	Proxy string `json:"proxy"`
	T0    int64  `json:"firstParse"`
	T1    int64  `json:"LastParse"`
}

type info1 struct {
	country  string
	ipOutput uint32
}

type info2 struct {
	mainPackageId uint16
	listPackage   []uint16
}

type proxyPackageRow struct {
	Id     uint16
	Name   string
	Url    string
	Load   int
	Use    int
	Weight int
}

type PaginatedIPPorts struct {
	Domain string       `json:"domain"`
	Data   []proxyParse `json:"data"`
}

type proxyParse struct {
	Ip       string `json:"ip"`
	Port     string `json:"port"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
}

type listInfo struct {
	total   uint
	unique  uint
	inCheck uint
}

type OutputPackageInfo struct {
	Id      uint16 `json:"id"`
	Name    string `json:"name"`
	Url     string `json:"url"`
	Total   uint   `json:"total"`
	Unique  uint   `json:"unique"`
	InCheck uint   `json:"inCheck"`
	Work    uint   `json:"work"`
}

type statsOut struct {
	CountPackage   uint                `json:"countPackage"`
	TotalAll       uint                `json:"totalAll"`
	TotalUniqueAll uint                `json:"totalUniqueAll"`
	TotalWork      uint                `json:"totalWork"`
	Packages       []OutputPackageInfo `json:"packages"`
}

type proxyRow struct {
	Datetime   int64 `json:"datetime"`
	proxy      string
	Country    string `json:"country"`
	IpOut      uint32 `json:"ipOut"`
	Latency    uint32 `json:"latency"`
	MaxLatency uint32 `json:"maxLatency"`
	Good       bool   `json:"good"`
	PackId     uint16 `json:"packId"`
}

type fullProxyInfo struct {
	Datatime    time.Time
	Proxy       string
	Country     string
	IpOutput    uint32
	latency     uint32
	MaxLatency  uint32
	good        bool
	packId      uint16
	HistLatency []int64
	HistGood    []string
	Disp        float64
}

type infoPerformance struct {
	ThreadsChecker int `json:"threadsChecker"`
	LenQueue       int `json:"lenQueue"`
}

const (
	_https = iota
	_socks4
	_socks5
)

type checkerRet struct {
	Proxy   string `json:"proxy"`
	IpOut   uint32 `json:"ipOut"`
	Country string `json:"country"`
	Latency uint32 `json:"latency"`
	Exist   bool   `json:"exist"`
}

type taskCheckerWeb struct {
	Hash       string       `json:"hash"`
	Total      uint32       `json:"total"`
	CountExist uint32       `json:"countExist"`
	Checked    uint32       `json:"checked"`
	Time       time.Time    `json:"time"`
	Proxylist  []checkerRet `json:"proxylist"`
	Status     bool         `json:"status"`
}
