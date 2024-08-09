package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type proxyInPack struct {
	PackageId uint16  `json:"packageId"`
	Proxy     string  `json:"proxy"`
	Good      bool    `json:"work"`
	Country   string  `json:"country"`
	IpOut     uint32  `json:"ipOut"`
	Disp      float64 `json:"dispersion"`
	History   []int64 `json:"history"`
	LastCheck int64   `json:"lastCheck"`
}

type responsePack struct {
	Total      uint          `json:"total"`
	Page       int           `json:"page"`
	MaxLatency int           `json:"maxLatency"`
	ProxyList  []proxyInPack `json:"proxyList"`
}

// Функция обработчик. Ожидает в запросе json от парсера. В ответ передает ошибки в строках и ok, если был создан пакет.
func ipPortsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	var ret = ""
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	defer r.Body.Close()
	var data PaginatedIPPorts
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !sanitizeDomain(data.Domain) {
		log.Printf("Character error in domain: %s", data.Domain)
		http.Error(w, "Character error", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./parsers", data.Domain+".txt")
	f := os.O_WRONLY | os.O_CREATE
	file, err := os.OpenFile(filePath, f, 0644)
	if err != nil {
		log.Printf("File open error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	var re = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	for _, proxy := range data.Data {

		t := protocol2byte(getProtocol(strings.ToLower(proxy.Protocol)))
		_, err = strconv.Atoi(proxy.Port)
		if len(re.FindStringIndex(proxy.Ip)) <= 0 {
			b, _ := json.Marshal(proxy)
			ret += fmt.Sprintf("Warning:  %s \n", b)
			continue
		} else if proxy.Port == "" {
			b, _ := json.Marshal(proxy)
			ret += fmt.Sprintf("Warning:  %s \n", b)
			continue
		} else if err != nil {
			b, _ := json.Marshal(proxy)
			ret += fmt.Sprintf("Warning: %s \n", b)
			continue
		}

		file.WriteString(fmt.Sprintf("%s:%s:%s:%s:%d\n", proxy.Ip, proxy.Port, proxy.Login, proxy.Password, t))
	}
	file.Close()
	ret += "Ok"
	w.Write([]byte(ret))
	updateProxyPackage(data.Domain)
	log.Printf("Обновлён пакет прокси: %s", data.Domain)
}

// Функция обработчик. Подготавливает json ответ о статистике по всем пакетам
func statisticApiHandler(w http.ResponseWriter, r *http.Request) {
	var all uint = 0
	var uniq uint = 0
	var count uint = 0
	var data = make([]OutputPackageInfo, 0)
	//var wp = getWorkProxies()
	//log.Println("l_api.go_1")
	mtx.Lock()
	//log.Println("l_api.go_2")
	PACKAGES.Range(func(k, v interface{}) bool {
		if v.(proxyPackageRow).Load != 0 {
			uniq += countProxy[k.(uint16)].unique
			pack := OutputPackageInfo{k.(uint16), v.(proxyPackageRow).Name, v.(proxyPackageRow).Url, countProxy[k.(uint16)].total, countProxy[k.(uint16)].unique, countProxy[k.(uint16)].inCheck, 0}
			data = append(data, pack)
			count++
		}
		return true
	})
	//log.Println("ul_api.go")
	mtx.Unlock()
	var st = statsOut{count, all, uniq, 0, data}
	w.Header().Set("Content-Type", "application/json")
	retJson, _ := json.Marshal(st)
	writeResponse(w, r, retJson)
}

// Функция обработчик. Подготавливает json ответ о статистике по определенному пакету.
func statsApiPackageHandler(w http.ResponseWriter, r *http.Request) {
	var InfoProxis []fullProxyInfo
	var idPack = r.URL.Query().Get("packageId")
	var id, err = strconv.Atoi(idPack)
	if err != nil {
		id = -1
	}
	InfoProxis = selectStatsForProxy(int16(id))
	var res responsePack
	res.Total = uint(len(InfoProxis))
	res.Page = 1
	res.MaxLatency = AppConfig.CheckerTimeoutSeconds * 1e6
	var pList = make([]proxyInPack, 0)
	for _, p := range InfoProxis {
		pList = append(pList, proxyInPack{p.packId, p.Proxy, p.good, p.Country, p.IpOutput, p.Disp, p.HistLatency, p.Datatime.Unix() * 1e3})
	}
	res.ProxyList = pList
	w.Header().Set("Content-Type", "application/json")
	retJson, err := json.Marshal(res)
	if err != nil {
		log.Println(err)
	}
	writeResponse(w, r, retJson)
}

// Отправляет JSON с информацией о количестве прокси в определенный момент времени
func graphWorkProxyApiHandler(w http.ResponseWriter, r *http.Request) {
	var workProxyL = selectCountWorkProxy()
	type toGraph struct {
		Total int   `json:"total"`
		Work  int   `json:"work"`
		Dtime int64 `json:"dateTime"`
	}
	var ans = make([]toGraph, 0)
	loc := time.FixedZone("", +3*60*60)
	for _, v := range workProxyL {
		tmp, _ := time.ParseInLocation("2006-1-02-15-04", v.D, loc)
		ans = append(ans, toGraph{v.Ct, v.Cw, tmp.Unix()})
	}
	retJson, _ := json.Marshal(ans)
	writeResponse(w, r, retJson)
}

func graphProxyApiHandler(w http.ResponseWriter, r *http.Request) {

}

func performanceApiHandler(w http.ResponseWriter, r *http.Request) {
	smtx.Lock()
	retJson, _ := json.Marshal(performanceThreads)
	smtx.Unlock()
	writeResponse(w, r, retJson)
}

func setSettingsApiHandler(w http.ResponseWriter, r *http.Request) {
	//log.Println("l_api.go_1")
	mtx.Lock()
	//log.Println("l_api.go_2")
	defer func() {
		//log.Println("ul_api.go")
		mtx.Unlock()
	}()
	var requestData map[string]interface{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for key, value := range requestData {
		switch key {
		case "requestsTimeToSend":
			AppConfig.RequestsTimeToSend = int(value.(float64))
		case "insertAttemptsRequests":
			AppConfig.InsertAttemptsRequests = int(value.(float64))
		case "histSecPerformanceThreads":
			AppConfig.HistSecPerformanceThreads = int(value.(float64))
		case "saveProxyDB":
			AppConfig.SaveProxyDB = int(value.(float64))
		case "checkerTimeoutSeconds":
			AppConfig.CheckerTimeoutSeconds = int(value.(float64))
		case "checkerStreamsMin":
			AppConfig.CheckerStreamsMin = int(value.(float64))
		case "checkerStreamsMax":
			AppConfig.CheckerStreamsMax = int(value.(float64))
		case "checkerVerbose":
			AppConfig.CheckerVerbose = value.(bool)
		case "checkerURL":
			AppConfig.CheckerURL = value.(string)
		case "loaderPeriodMinutes":
			AppConfig.LoaderPeriodMinutes = int(value.(float64))
		case "checkerPackagesSecond":
			AppConfig.CheckerPackagesSecond = int(value.(float64))
		}
	}

	// Save to file if requested
	if saveToFile, ok := requestData["saveToFile"].(bool); ok && saveToFile {
		file, err := os.Create("conf.json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(AppConfig); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func getSettingsApiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(AppConfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getWorkProxy() {

}

func instantCheckListApiHandler(w http.ResponseWriter, r *http.Request) {
	type ProxyRequest struct {
		Proxies []string `json:"proxies"`
	}
	var proxy2Check = make([]proxy, 0)
	currentTime := time.Now()
	timeString := currentTime.Format(time.RFC3339)
	hash := sha256.New()
	hash.Write([]byte(timeString))
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	var proxyRequest ProxyRequest
	err := json.NewDecoder(r.Body).Decode(&proxyRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var proxyList = make([]checkerRet, 0)
	var total uint32 = 0
	for _, s := range proxyRequest.Proxies {
		var p proxy
		p, err = parseProxy(s)
		if err != nil && err.Error() != "Proxy already exist" {
			continue
		}
		var idx = slices.IndexFunc(proxy2Check, func(elt proxy) bool { return elt == p })
		if err != nil && err.Error() == "Proxy already exist" {
			//log.Println("l_api.go_1")
			mtx.Lock()
			//log.Println("l_api.go_2")
			v, ok := checkerInfo1[p]
			//log.Println("ul_api.go")
			mtx.Unlock()
			if idx < 0 {
				total++
			}
			if ok && v.ipOutput != 0 {
				proxyList = append(proxyList, checkerRet{proxy2String(p), v.ipOutput, v.country, 0, true})
			}
			continue
		}
		if idx < 0 {
			proxy2Check = append(proxy2Check, p)
		}

	}
	//log.Println("l_api.go_1")
	mtx.Lock()
	//log.Println("l_api.go_2")
	tasksCheckerWeb[hashString] = taskCheckerWeb{hashString, uint32(len(proxy2Check)) + total, total, total, currentTime, proxyList, false}
	//log.Println("ul_api.go")
	mtx.Unlock()
	for _, p := range proxy2Check {
		go checkerProxyWeb(p, hashString)
	}
	var mapHash = make(map[string]string)
	mapHash["id"] = hashString
	retJson, err := json.Marshal(mapHash)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	writeResponse(w, r, retJson)
	go func(hashString string) {
		time.Sleep(time.Duration(15) * time.Minute)
		//log.Println("l_api.go_1")
		mtx.Lock()
		//log.Println("l_api.go_2")
		delete(tasksCheckerWeb, hashString)
		//log.Println("ul_api.go")
		mtx.Unlock()
	}(hashString)
}

func getResultWebCheckerApiHandler(w http.ResponseWriter, r *http.Request) {
	var hash = r.URL.Query().Get("hash")
	if hash == "" {
		w.Write([]byte("{}"))
		return
	}
	var ret, ok = tasksCheckerWeb[hash]
	if !ok {
		w.Write([]byte("{}"))
		return
	}
	if ret.Checked == ret.Total {
		ret.Status = true
		tasksCheckerWeb[hash] = ret
	}
	var retJson, _ = json.Marshal(ret)
	w.Header().Set("Content-Type", "application/json")
	writeResponse(w, r, retJson)
	return
}
