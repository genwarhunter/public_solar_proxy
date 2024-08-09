package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func ip2long(ipAddress string) uint32 {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip.To4())
}

func long2ip(ipAddress uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, ipAddress)
	return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
}

func httpGET(url string, maxAttempts int) string {
	var ans strings.Builder
	var a, b, c, d, p int
	if TEST_RANDOM_PROXY {
		for i := 0; i < TEST_RANDOM_PROXIES_PER_PKG; i++ {
			a = rand.Int() % 255
			b = rand.Int() % 255
			c = rand.Int() % 255
			d = rand.Int() % 255
			p = rand.Int() % 65535
			_, _ = fmt.Fprintf(&ans, "%d.%d.%d.%d:%d\n", a, b, c, d, p)
		}
		return ans.String()
	}

	var attempt = 0
	var resp *http.Response
	var err error
	for attempt < maxAttempts {
		attempt++
		resp, err = http.Get(url)
		if err != nil || resp.StatusCode != 200 {
			time.Sleep(20 * time.Second)
			continue
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			return string(b)
		}
	}
	return ""
}

// возвращает ip адрес первого попавшегося интерфейса (кроме lo)
func getExternalIP() string {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range interfaces {
		addresses, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addresses {
			var q []string
			q = strings.Split(a.String(), "/")
			if len(q) != 2 || len(strings.Split(q[0], ".")) != 4 {
				continue
			}
			ip := net.ParseIP(q[0])
			if ip == nil {
				continue
			}
			if ip.IsPrivate() || ip.IsLoopback() {
				continue
			}
			ips = append(ips, q[0])
		}
	}
	if len(ips) == 0 {
		log.Fatalf("не определена переменная checkerURL и не удалось определить внешний IP адрес")
	}
	if len(ips) > 1 {
		log.Fatalf("не определена переменная checkerURL и найдено несколько внешних IP адресов")
	}
	return ips[0]
}

// удаляет из sync.Map ключи, не перечисленные в keysToKeep
func removeOtherKeys(keysToKeep map[interface{}]bool, m *sync.Map) {
	var allKeys []interface{}
	m.Range(func(k interface{}, v interface{}) bool {
		allKeys = append(allKeys, k)
		return true
	})
	for _, k := range allKeys {
		if !keysToKeep[k] {
			m.Delete(k)
		}
	}
}

func getProtocol(st string) string {
	if strings.Contains(st, "http") {
		return "http"
	}
	if strings.Contains(st, "socks4") {
		return "socks4"
	}
	if strings.Contains(st, "socks5") {
		return "socks5"
	}
	return ""
}

func protocol2byte(t string) byte {
	switch t {
	case "https":
		return _https
	case "http":
		return _https
	case "socks4":
		return _socks4
	case "socks5":
		return _socks5
	default:
		return _https
	}
}

func byte2protocol(t byte) string {
	switch t {
	case _https:
		return "http"
	case _socks4:
		return "socks4"
	case _socks5:
		return "socks5"
	default:
		return ""
	}
}

func int2bool(i int) bool {
	return i == 1
}

// при удалении значений из словаря, памать не освобождаетя
// https://github.com/golang/go/issues/20135
// (единственное ?) решение - пересоздать словари
//func recreateGlobalMaps() {
//	for {
//		time.Sleep(RECREATE_SLEEP)
//		mtx.Lock()
//		////log.Println("l_helper.go")
//		var oldGC = debug.SetGCPercent(-1)
//		{
//			var newMap = make(map[proxy]int)
//			for k, v := range UsageCount {
//				newMap[k] = v
//			}
//			UsageCount = newMap
//		}
//		debug.FreeOSMemory()
//		debug.SetGCPercent(oldGC)
//		mtx.Unlock()
//		////log.Println("ul_helper.go")
//	}
//}

// true - в столбце load таблицы package поставлено 1, иначе false
func pkgIsLoad(pkgId uint16) bool {
	var val, ok = PACKAGES.Load(pkgId)
	if !ok {
		return false
	}
	return val.(proxyPackageRow).Load == 1
}

// true - в столбце use таблицы package поставлено 1, иначе false
func pkgIsUse(pkgId int) bool {
	var val, ok = PACKAGES.Load(pkgId)
	if !ok {
		return false
	}
	return val.(proxyPackageRow).Use == 1
}

// TODO: Исправить
func isWork(p proxy) bool {
	var ret = false
	//_, ok := checkerHistory[p]
	//if !ok {
	//	return false
	//}
	//var l = len(checkerHistory[p])
	//var ret = checkerHistory[p][l-1].Good
	//var lim = min(l, checkerMaxFails)
	//for i := 2; i <= lim; i++ {
	//	ret = ret || checkerHistory[p][l-i].Good
	//}
	return ret
}

func compressBytes(in []byte) *[]byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(in); err != nil {
		fmt.Println(err)
		return nil
	}
	if err := gz.Close(); err != nil {
		fmt.Println(err)
		return nil
	}
	ans := b.Bytes()
	return &ans
}

func parseProxy(proxyString string) (proxy, error) {
	// Регулярное выражение для проверки и извлечения IP и порта
	re := regexp.MustCompile(`^(?:(https?|socks[45])://)?(?:([^:]*):([^@]*)@)?([^:\/]+):(\d+)$`)
	match := re.FindStringSubmatch(proxyString)

	if match == nil {
		return proxy{}, fmt.Errorf("invalid proxy format")
	}

	protocolS := match[1]
	if protocolS == "" {
		protocolS = ""
	}
	login := match[2]
	if login == "" {
		login = ""
	}
	password := match[3]
	if password == "" {
		password = ""
	}
	ip := match[4]
	portS := match[5]
	port, _ := strconv.Atoi(portS)
	var p = proxy{ip2long(ip), uint16(port)}

	protocol := protocol2byte(protocolS)
	////log.Println("l_helper.go_1")
	mtx.Lock()
	////log.Println("l_helper.go_2")
	defer mtx.Unlock()
	_, ok := infoProxy[p]
	if ok {
		return p, fmt.Errorf("Proxy already exist")
	}

	if protocolS != "" {
		infoProxy[p] = proxyInfo{login, password, protocol}
	}

	return p, nil
}

func writeResponse(w http.ResponseWriter, r *http.Request, retJson []byte) {
	p := &retJson
	if checkCompress(r.Header) {
		compressed := compressBytes(retJson)
		if compressed != nil {
			w.Header().Set("Content-Encoding", "gzip")
			p = compressed
		}
		w.Write(*p)
		return
	}
	_, _ = w.Write(retJson)
}

func proxy2String(p proxy) string {
	var proxyStr = byte2protocol(infoProxy[p].protocol) + "://"
	if infoProxy[p].login != "" {
		proxyStr += infoProxy[p].login + ":" + infoProxy[p].password + "@"
	}
	proxyStr += long2ip(p.ip) + ":" + strconv.Itoa(int(p.port))
	return proxyStr
}

func string2Proxy(proxystr string) (proxy, proxyInfo) {
	//http://ttk1KC:7Np2bu@177.234.140.177:8000
	var s = strings.Split(proxystr, "@")
	var info proxyInfo
	var p proxy
	if len(s) > 1 {
		plp := strings.Split(s[0], "://")
		protocol := protocol2byte(plp[0])
		lp := strings.Split(plp[1], ":")
		info = proxyInfo{lp[0], lp[1], protocol}
		ipPort := strings.Split(s[1], ":")
		ip := ip2long(ipPort[0])
		port, _ := strconv.Atoi(ipPort[1])
		p = proxy{ip, uint16(port)}
	} else {
		pip := strings.Split(s[0], "://")
		ipPort := strings.Split(pip[1], ":")
		ip := ip2long(ipPort[0])
		port, _ := strconv.Atoi(ipPort[1])
		p = proxy{ip, uint16(port)}
		protocol := protocol2byte(pip[0])
		info = proxyInfo{"", "", protocol}
	}

	return p, info
}

func checkCompress(h http.Header) bool {
	acceptGzip := false

	for k, v := range h {
		for _, vv := range v {
			if k == "Accept-Encoding" && strings.Contains(vv, "gzip") {
				acceptGzip = true
			}
		}
	}
	return acceptGzip
}

func StringToDate(date string) time.Time {
	parsed, err := time.Parse("2006-01-02 15:04:05", date)
	if err != nil {
		panic(err)
	}
	return parsed
}

func sanitizeDomain(domain string) bool {
	// Add more checks as necessary
	if strings.ContainsAny(domain, "/\\<>:\"|?*") {
		return false
	}
	return true
}
