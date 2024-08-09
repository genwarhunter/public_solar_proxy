package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var smtx sync.RWMutex
var performanceThreads map[int64]infoPerformance
var threadsNow atomic.Int32
var threadsLimit int
var memUsageMB uint64

var mtx sync.RWMutex
var quit chan struct{}
var queue chan proxy

var dontcheck map[proxy]struct{}
var loaderUniq map[proxy]info2
var infoProxy map[proxy]proxyInfo
var checkerInfo1 map[proxy]info1

var toRemove []proxy

var countProxy map[uint16]listInfo
var toClickHouse []proxyRow

var tasksCheckerWeb map[string]taskCheckerWeb

func refresh() {
	firstIteration := true
	for {
		var arr, ok = selectFromPackage()
		if ok {
			pkgIds := make(map[interface{}]bool)
			for _, p := range arr {
				pkgIds[p.Id] = true
				PACKAGES.Store(p.Id, p)
			}
			removeOtherKeys(pkgIds, &PACKAGES)
			if firstIteration {
				firstIteration = false
			}
		}
		time.Sleep(REFRESH_SLEEP)
	}
}

func updateMemUsage() {
	var m runtime.MemStats
	for {
		time.Sleep(time.Second * 30)
		runtime.GC()
		runtime.ReadMemStats(&m)
		//log.Println("l_main.go_1")
		mtx.Lock()
		//log.Println("l_main.go_2")
		memUsageMB = m.Alloc / (1024 * 1024)
		fmt.Printf("%d MB\n", memUsageMB)
		//log.Println("ul_main.go")
		mtx.Unlock()
	}
}

func init() {
	getConfig()
	rand.Seed(time.Now().Unix())
	threadsNow.Store(0)
	quit = make(chan struct{}, 1)
	threadsLimit = AppConfig.CheckerStreamsMin
	queue = make(chan proxy, 500000)
	chInsert = make(chan bool, 1)
	performanceThreads = make(map[int64]infoPerformance)
	loaderUniq = make(map[proxy]info2)
	checkerInfo1 = make(map[proxy]info1)
	infoProxy = make(map[proxy]proxyInfo)
	dontcheck = make(map[proxy]struct{})
	countProxy = make(map[uint16]listInfo)
	toRemove = make([]proxy, 0)
	tasksCheckerWeb = make(map[string]taskCheckerWeb)
	InitGeoIpDB()
	createConnectMysql()
	loadDontCheck()
	if AppConfig.CheckerURL == "" {
		var ip = getExternalIP()
		var i = strings.Split(AppConfig.ListenAddr, ":")
		if len(i) == 2 {
			_, err := strconv.ParseUint(i[1], 10, 16)
			if err == nil {
				AppConfig.CheckerURL = fmt.Sprintf("http://%s:%s/api/v1/checker/ping", ip, i[1])
				fmt.Println("checkerURL =", AppConfig.CheckerURL)
			}
		}
	}
}

func main() {
	go refresh()
	go threads()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.Handle("/parsers/", http.StripPrefix("/parsers/", http.FileServer(http.Dir("./parsers"))))

	http.HandleFunc("/api/v2/fromParser", ipPortsHandler)

	http.HandleFunc("/api/v1/checker/ping", checkerHandler)

	http.HandleFunc("/api/v2/setConfig", setSettingsApiHandler)
	http.HandleFunc("/api/v2/getConfig", getSettingsApiHandler)
	http.HandleFunc("/settings", settings)

	http.HandleFunc("/api/v2/instantCheckList", instantCheckListApiHandler)
	http.HandleFunc("/checker", instantCheckList)

	http.HandleFunc("/api/v2/getResultWebChecker", getResultWebCheckerApiHandler)
	http.HandleFunc("/checker/result/", getResultWebChecker)

	http.HandleFunc("/api/v2/stats", statisticApiHandler)
	http.HandleFunc("/stats", statisticHandler)

	http.HandleFunc("/api/v2/graphsWork", graphWorkProxyApiHandler)
	http.HandleFunc("/graphs", renderGraphsStat)
	http.HandleFunc("/api/v2/checkProxyNew", graphProxyApiHandler)

	http.HandleFunc("/api/v2/statsPackage", statsApiPackageHandler)
	http.HandleFunc("/package", statsPackageHandler)

	http.HandleFunc("/api/v2/performance", performanceApiHandler)
	http.HandleFunc("/performance", renderGraphPerformance)

	//http.HandleFunc("/api/v2/getWorkProxy", workApiHandler)
	//http.HandleFunc("/work", workList)
	//http.HandleFunc("/getWorkProxy", workProxy)

	log.Fatal(http.ListenAndServe(AppConfig.ListenAddr, nil))
}
