package main

import (
	"fmt"
	"h12.io/socks"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const checkerLog = "чекер"
const checkerMaxFails = 3

// Поток для проверки прокси.
// Запускается при необходимости из функции checkProxies.
func checkProxy() {
	defer func() {
		threadsNow.Add(-1)
	}()
	for {
		select {
		case <-quit:
			return
		default:
			var p = <-queue
			var country = ""
			var ipOut uint32 = 0
			var good = false
			var checkerResponse, latency, timeCheck = getRequestWithProxy(p)
			var z = strings.Split(checkerResponse, ":")
			if len(z) != 2 || ip2long(z[0]) == 0 {
				if AppConfig.CheckerVerbose {
					//log.Println("l_checker.go_1")
					mtx.Lock()
					//log.Println("l_checker.go_2")
					log.Printf("%s пакет %d прокси:%s:%d  неверный ответ:%s", checkerLog, loaderUniq[p].mainPackageId, long2ip(p.ip), p.port, checkerResponse)
					//log.Println("ul_checker.go")
					mtx.Unlock()
				}
			} else {
				good = true
				ipOut = ip2long(z[0])
				country = getCountryByIp(z[0])
				if AppConfig.CheckerVerbose {
					//log.Println("l_checker.go_1")
					mtx.Lock()
					//log.Println("l_checker.go_2")
					log.Printf("%s пакет:%d %s %s:%d -> %s '%s'", checkerLog, loaderUniq[p].mainPackageId, byte2protocol(infoProxy[p].protocol), long2ip(p.ip), p.port, z[0], country)
					//log.Println("ul_checker.go")
					mtx.Unlock()
				}
			}
			//log.Println("l_checker.go_1")
			mtx.Lock()
			//log.Println("l_checker.go_2")
			checkerInfo1[p] = info1{country, ipOut}
			var tmp = proxyRow{timeCheck, proxy2String(p), country, ipOut, uint32(latency), uint32((time.Second * time.Duration(AppConfig.CheckerTimeoutSeconds)).Microseconds()), good, loaderUniq[p].mainPackageId}
			toClickHouse = append(toClickHouse, tmp)
			//log.Println("ul_checker.go")
			mtx.Unlock()
			go func(good bool) {
				//log.Println("l_checker.go_1")
				mtx.Lock()
				//log.Println("l_checker.go_2")
				var _, ok = dontcheck[p]
				if ok && !good {
					countProxy[loaderUniq[p].mainPackageId] = listInfo{countProxy[loaderUniq[p].mainPackageId].total, countProxy[loaderUniq[p].mainPackageId].unique, countProxy[loaderUniq[p].mainPackageId].inCheck - 1}
					//log.Println("ul_checker.go")
					mtx.Unlock()
					return
				} else if ok && good {
					delete(dontcheck, p)
					toRemove = append(toRemove, p)
					countProxy[loaderUniq[p].mainPackageId] = listInfo{countProxy[loaderUniq[p].mainPackageId].total, countProxy[loaderUniq[p].mainPackageId].unique, countProxy[loaderUniq[p].mainPackageId].inCheck + 1}
				}
				//log.Println("ul_checker.go")
				mtx.Unlock()
				var m = time.Duration(AppConfig.CheckerTimeoutSeconds)*time.Second - time.Duration(latency/1e6)
				if m > 0 {
					time.Sleep(m)
				}
				queue <- p
			}(good)
		}
	}
}

// Поток информации о производительности
func perfomansStats() {
	var v0 int32 = 0
	for {
		var t = threadsNow.Load()
		now := time.Now().Unix()
		////log.Println("l_ckecker.go")
		smtx.Lock()
		performanceThreads[now] = infoPerformance{int(t), len(queue)}
		for k, _ := range performanceThreads {
			if k < now-int64(AppConfig.HistSecPerformanceThreads) {
				delete(performanceThreads, k)
			}
		}
		////log.Println("ul_checker.go")
		smtx.Unlock()
		if t != v0 {
			fmt.Println("Горутин", t-v0, t)
		}
		v0 = t
		time.Sleep(700 * time.Millisecond)
	}
}

// Запуск потоков чекера для проверки прокси.
func startThreadsChecker() {
	for {
		var t = threadsNow.Load()
		var queueLen = len(queue)
		threadsLimit = min(max(AppConfig.CheckerStreamsMin, queueLen/10), AppConfig.CheckerStreamsMax)
		if t < int32(threadsLimit) {
			for i := 0; i < threadsLimit-int(t); i++ {
				threadsNow.Add(1)
				go checkProxy()
			}
		}

		if t > int32(threadsLimit) {
			for i := 0; i < int(t)-threadsLimit; i++ {
				go func() {
					quit <- struct{}{}
				}()
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func checkerProxyWeb(p proxy, hashString string) {
	threadsNow.Add(1)
	defer threadsNow.Add(-1)
	////log.Println("l_ckecker.go")
	//log.Println("l_checker.go_1")
	mtx.Lock()
	//log.Println("l_checker.go_2")
	_, ok := infoProxy[p]
	//log.Println("ul_checker.go")
	mtx.Unlock()
	var response string
	var latency int64
	if !ok {
		//log.Println("l_checker.go_1")
		mtx.Lock()
		//log.Println("l_checker.go_2")
		infoProxy[p] = proxyInfo{"", "", 0}
		////log.Println("ul_checker.go")
		mtx.Unlock()
		response, latency, _ = getRequestWithProxy(p)
		if response == "" {
			//log.Println("l_checker.go_1")
			mtx.Lock()
			//log.Println("l_checker.go_2")
			infoProxy[p] = proxyInfo{"", "", 1}
			//log.Println("ul_checker.go")
			mtx.Unlock()
			response, latency, _ = getRequestWithProxy(p)
			if response == "" {
				//log.Println("l_checker.go_1")
				mtx.Lock()
				//log.Println("l_checker.go_2")
				infoProxy[p] = proxyInfo{"", "", 2}
				//log.Println("ul_checker.go")
				mtx.Unlock()
				response, latency, _ = getRequestWithProxy(p)
			}
		}
	} else {
		response, latency, _ = getRequestWithProxy(p)
	}
	var z = strings.Split(response, ":")
	var ipOut uint32
	var country string
	//log.Println("l_checker.go_1")
	mtx.Lock()
	//log.Println("l_checker.go_2")
	var task = tasksCheckerWeb[hashString]
	if len(z) == 2 && ip2long(z[0]) != 0 {
		ipOut = ip2long(z[0])
		country = getCountryByIp(z[0])
		task.Proxylist = append(task.Proxylist, checkerRet{proxy2String(p), ipOut, country, uint32(latency), false})
	}
	checkerInfo1[p] = info1{country, ipOut}
	task.Checked++
	if task.Checked == task.Total {
		task.Status = true
	}
	tasksCheckerWeb[hashString] = task
	//log.Println("l_ckecker.go")
	mtx.Unlock()
}

// Проверка прокси. На вход принимает прокси.
// Возвращает ответ от прокси, задержку и время проверки.
// Если прокси не рабочее, возвращает пустой ответ, таймаут и время выхода из функции.
func getRequestWithProxy(p proxy) (string, int64, int64) {
	defer func() { recover() }()
	if TEST_CHECKER_ALWAYS_OK || TEST_RANDOM_PROXY {
		time.Sleep(TEST_CHECKER_SLEEP)
		return fmt.Sprintf("%s:%d", long2ip(p.ip), p.port), int64(time.Duration(AppConfig.CheckerTimeoutSeconds) * time.Second), time.Now().Unix()
	}
	var requestUrl, _ = url.Parse(AppConfig.CheckerURL)
	var transport = &http.Transport{}
	var proxyUrl *url.URL
	var err error
	////log.Println("l_ckecker.go")
	mtx.RLock()
	proxyUrl, err = url.Parse(proxy2String(p))
	////log.Println("ul_checker.go")
	mtx.RUnlock()
	if err != nil {
		log.Printf(err.Error())
		return "", int64(int(time.Duration(AppConfig.CheckerTimeoutSeconds) * time.Second)), time.Now().Unix()
	}
	mtx.RLock()
	if infoProxy[p].protocol == _https {
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
	} else {
		dial := socks.Dial(proxyUrl.String())
		transport = &http.Transport{Dial: dial}
	}
	mtx.RUnlock()
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(AppConfig.CheckerTimeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		if AppConfig.CheckerVerbose {
			log.Printf("%s: %s:%d - %s", checkerLog, long2ip(p.ip), p.port, err)
		}
		return "", int64(time.Duration(AppConfig.CheckerTimeoutSeconds) * time.Second), time.Now().Unix()
	}

	start := time.Now()
	response, err := client.Do(request)
	elapsed := time.Now().Sub(start)

	if err != nil {
		if AppConfig.CheckerVerbose {
			log.Printf("%s: %s:%d - %s", checkerLog, long2ip(p.ip), p.port, err)
		}
		return "", elapsed.Microseconds(), start.Unix()
	}
	defer func() {
		_ = response.Body.Close()
		client.CloseIdleConnections()
	}()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		if AppConfig.CheckerVerbose {
			log.Printf("%s: %s:%d - %s", checkerLog, long2ip(p.ip), p.port, err)
		}
		return "", elapsed.Microseconds(), start.Unix()
	}
	return string(data), elapsed.Microseconds(), start.Unix()
}

func checkerHandler(w http.ResponseWriter, r *http.Request) {
	for k, v := range r.Header {
		for _, vv := range v {
			if k == "X-Real-Ip" {
				_, _ = fmt.Fprintf(w, "%s:0\n", vv)
				return
			}
		}
	}
	_, _ = fmt.Fprintf(w, "%s\n", r.RemoteAddr)
}
