package main

import (
	"log"
	"slices"
	"strconv"
	"strings"
)

const loaderTag = "загрузка списка прокси"

func loadProxyList(pkgId uint16) {
	var localUniq = make(map[proxy]bool)
	var toCheck uint
	var val, ok = PACKAGES.Load(pkgId)
	var pkg = val.(proxyPackageRow)
	var add uint
	if !ok {
		return
	}
	proxyPackageResponse := httpGET(pkg.Url, 3)
	if proxyPackageResponse == "" {
		log.Printf("%s %d:%s ошибка запроса url:%s", loaderTag, pkg.Id, pkg.Name, pkg.Url)
		return
	}

	proxyPackageResponse = strings.Replace(proxyPackageResponse, "\r", "", -1)
	rows := strings.Split(proxyPackageResponse, "\n")
	if len(rows) == 0 {
		log.Printf("%s %d:%s не найдены прокси в ответе url:%s", loaderTag, pkg.Id, pkg.Name, pkg.Url)
		return
	}
	//log.Println("l_loader.go_1")
	mtx.Lock()
	//log.Println("l_loader.go_2")
	defer func() {
		//log.Println("ul_loader.go")
		mtx.Unlock()
	}()
	for _, row := range rows {
		if len(row) > 0 {
			ipPort := strings.Split(row, ":")
			if len(ipPort) != 2 && len(ipPort) != 5 {
				//log.Printf("%s %d:%s неправильная строка: %s Строка %d из %d", loaderTag, pkg.Id, pkg.Name, row, i, len(rows))
				continue
			}
			ip := ip2long(ipPort[0])
			port, err := strconv.Atoi(ipPort[1])
			if err != nil || ip == 0 {
				//log.Printf("%s %d:%s неправильная строка: %s Строка %d из %d", loaderTag, pkg.Id, pkg.Name, row, i, len(rows))
				continue
			}
			if TEST_RANDOM_PROXY {
				continue
			}
			var p = proxy{ip, uint16(port)}
			if len(ipPort) == 2 {
				if strings.Contains(pkg.Url, "http.txt") {
					infoProxy[p] = proxyInfo{"", "", _https}
				} else if strings.Contains(pkg.Url, "https.txt") {
					infoProxy[p] = proxyInfo{"", "", _https}
				} else if strings.Contains(pkg.Url, "socks4") {
					infoProxy[p] = proxyInfo{"", "", _socks4}
				} else if strings.Contains(pkg.Url, "socks5") {
					infoProxy[p] = proxyInfo{"", "", _socks5}
				} else {
					infoProxy[p] = proxyInfo{"", "", _https}
				}
			} else if len(ipPort) == 5 {
				var t, _ = strconv.Atoi(ipPort[4])
				infoProxy[p] = proxyInfo{ipPort[2], ipPort[3], byte(t)}
			}
			_, ok1 := localUniq[p]
			if ok1 {
				continue
			}
			var oldPkg, ok2 = loaderUniq[p]
			var _, ok3 = dontcheck[p]

			if !ok2 && !ok3 {
				toCheck++
				queue <- p
				newPkg := info2{mainPackageId: pkgId}
				newPkg.listPackage = append(newPkg.listPackage, pkgId)
				loaderUniq[p] = newPkg
			} else {
				if !slices.Contains(loaderUniq[p].listPackage, pkgId) {
					add++
					tmp := append(loaderUniq[p].listPackage, pkgId)
					loaderUniq[p] = info2{oldPkg.mainPackageId, tmp}
				}
			}

		}
	}

	countProxy[pkgId] = listInfo{countProxy[pkgId].total + toCheck + add, countProxy[pkgId].unique + toCheck, countProxy[pkgId].inCheck + toCheck}
	//log.Printf("%s, пакет %d:%s, изменение +%d-%d=%d, итого %d, количество уникальных записей: %d\n", loaderTag, pkg.Id, pkg.Name, newCount, deleteCount, newCount-deleteCount, len(ans), len(localUniq))
}
