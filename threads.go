package main

import (
	"log"
	"sync"
	"time"
)

// PACKAGES вся информация о пакетах
var PACKAGES sync.Map             // map[int]proxyPackageRow
var PackageThreadRunning sync.Map // для каких пакетов запущены потоки чекера

func isPkgThreadRunning(pkgId uint16) bool {
	var val, ok = PackageThreadRunning.Load(pkgId)
	return ok && val.(bool)
}

func threadForPackage(pkg proxyPackageRow) {
	log.Printf("запуск чекера для пакета %d:%s", pkg.Id, pkg.Name)
	defer func() {
		var arr []proxy
		//log.Println("l_threads.go_1")
		mtx.Lock()
		//log.Println("l_threads.go_2")
		for k, info := range loaderUniq {
			if info.mainPackageId == pkg.Id {
				arr = append(arr, k)
			}
		}
		delete(countProxy, pkg.Id)
		//log.Println("ul_threads.go")
		mtx.Unlock()
		log.Printf("чекер для пакета %d:%s остановлен, удалено прокси: %d", pkg.Id, pkg.Name, len(arr))
		PackageThreadRunning.Store(pkg.Id, false)
	}()

	var _ []proxy
	var lastLoad, _ time.Time
	var _, checkCount = 0, 0
	for {
		if !pkgIsLoad(pkg.Id) {
			break
		}
		if checkCount == 0 || time.Since(lastLoad) > time.Duration(AppConfig.LoaderPeriodMinutes)*time.Minute {
			loadProxyList(pkg.Id)
			lastLoad = time.Now()
			checkCount++
		}
		time.Sleep(1 * time.Second)
	}
}

func checkPackage() {
	//log.Println("l_threads.go_1")
	mtx.Lock()
	//log.Println("l_threads.go_2")
	defer func() {
		//log.Println("ul_threads.go")
		mtx.Unlock()
	}()
	change := 0
	for p, pack := range loaderUniq {
		var m uint = 0
		newPkgId := loaderUniq[p].mainPackageId
		oldPkgId := loaderUniq[p].mainPackageId
		for _, pid := range pack.listPackage {
			if oldPkgId == 1 || oldPkgId == 2 || oldPkgId == 3 || countProxy[pid].total > m {
				newPkgId = pid
				m = countProxy[pid].total
			}
		}

		if oldPkgId != newPkgId && (newPkgId != 1 && newPkgId != 2 && newPkgId != 3) {
			change++
			var tmp1 = countProxy[oldPkgId]
			var tmp2 = countProxy[newPkgId]
			_, ok := loaderUniq[p]

			var add uint = 0
			if oldPkgId == 1 || oldPkgId == 2 || oldPkgId == 3 {
				add = 1
			}
			if tmp1.inCheck == 0 || tmp1.unique == 0 {
				continue
			}
			if ok {
				tmp1 = listInfo{tmp1.total - add, tmp1.unique - 1, tmp1.inCheck - 1}
				tmp2 = listInfo{tmp2.total + add, tmp2.unique + 1, tmp2.inCheck + 1}
			} else {
				tmp1 = listInfo{tmp1.total - add, tmp1.unique - 1, tmp1.inCheck}
				tmp2 = listInfo{tmp2.total + add, tmp2.unique + 1, tmp2.inCheck}
			}
			countProxy[oldPkgId] = tmp1
			countProxy[newPkgId] = tmp2
			loaderUniq[p] = info2{mainPackageId: newPkgId, listPackage: loaderUniq[p].listPackage}
		}
	}

	if change > 0 {
		log.Printf("checkPackage Изменений: %d", change)
	}
}

func safeProxy() {
	for {
		time.Sleep(time.Minute * time.Duration(AppConfig.SaveProxyDB))
		var proxyL []proxy
		//log.Println("l_threads.go_1")
		mtx.Lock()
		//log.Println("l_threads.go_2")
		defer func() {
			//log.Println("ul_threads.go")
			mtx.Unlock()
		}()
		for k := range loaderUniq {
			proxyL = append(proxyL, k)
		}
		for k := range dontcheck {
			proxyL = append(proxyL, k)
		}
		copiedProxyL := make([]proxy, len(proxyL))
		copy(copiedProxyL, proxyL)
		////log.Println("ul_threads.go4")
		mtx.Unlock()
		insertProxy(copiedProxyL)
	}

}

func loadProxyFromDB() {
	for {
		var plist = make([]proxy, 0)
		var infoList = make([]proxyInfo, 0)
		var idList = make([]uint16, 0)
		plist, infoList, idList = selectAllProxy()
		//log.Println("l_threads.go_1")
		mtx.Lock()
		//log.Println("l_threads.go_2")
		for i, p := range plist {
			infoProxy[p] = infoList[i]
			var tmp []uint16
			tmp = append(tmp, idList[i])
			_, ok := loaderUniq[p]
			_, ok2 := dontcheck[p]
			if !ok && !ok2 {
				countProxy[idList[i]] = listInfo{countProxy[idList[i]].total + 1, countProxy[idList[i]].unique + 1, countProxy[idList[i]].inCheck + 1}
				loaderUniq[p] = info2{idList[i], tmp}
				queue <- p
			}
		}
		//log.Println("ul_threads.go")
		mtx.Unlock()
		time.Sleep(3 * time.Hour)
	}

}

func threads() {
	var lastCheck = time.Now()
	go updateMemUsage()
	go insertRequestsIfTime()
	go safeProxy()
	go perfomansStats()
	go startThreadsChecker()
	go updateDontCheck()
	go loadProxyFromDB()
	for {
		PACKAGES.Range(func(key, value interface{}) bool {
			var pkgId = key.(uint16)
			var pkgRow = value.(proxyPackageRow)
			if pkgRow.Load != 1 {
				return true
			}
			if !isPkgThreadRunning(pkgId) {
				PackageThreadRunning.Store(pkgRow.Id, true)
				go threadForPackage(pkgRow)
			}
			return true
		})

		if time.Since(lastCheck) > time.Duration(AppConfig.CheckerPackagesSecond)*time.Second {
			checkPackage()
			lastCheck = time.Now()
		}
		time.Sleep(5 * time.Second)
	}
}

func loadDontCheck() {
	var pArr = make([]proxy, 0)
	var packs = make([]uint16, 0)
	var tmp = make(map[uint16]uint)
	pArr, _, packs = selectDontCheckProxy()
	for i, p := range pArr {
		dontcheck[p] = struct{}{}
		tmp[packs[i]]++
	}
	for k, v := range tmp {
		countProxy[k] = listInfo{v, v, 0}
	}
	log.Printf("Загружено нерабочих прокси: %d", len(pArr))
}

func updateDontCheck() {
	for {
		time.Sleep(1 * time.Hour)
		var plist = make([]proxy, 0)
		plist = getLastHoursNotWork()
		//log.Println("l_threads.go_1")
		mtx.Lock()
		//log.Println("l_threads.go_2")
		for _, p := range plist {
			dontcheck[p] = struct{}{}
		}
		removeFromDontCheck()
		//log.Println("ul_threads.go")
		mtx.Unlock()
		insertDontCheck()
		log.Println("updateDontCheck: ", len(plist))
	}
}
