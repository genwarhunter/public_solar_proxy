package main

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"log"
	"time"
)

var dbAddr1, dbName1, dbUserName1, dbPassword1 string

var insertSleep = 30

var chInsert chan bool
var inserts int

type statsWotk struct {
	Proxy      string  `json:"proxy"`
	Percent    float32 `json:"percent"`
	LenHistory uint16  `json:"lenHistory"`
	AvgLatency uint32  `json:"avgLatency"`
}

func createConnectClickhouse() clickhouse.Conn {
	var clickhouseConnect, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{AppConfig.ClickhouseAddr},
		Auth: clickhouse.Auth{
			Database: AppConfig.ClickhouseDB,
			Username: AppConfig.ClickhouseUser,
			Password: AppConfig.ClickhousePassword,
		},
		Debug:           false,
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    100,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionNone,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
	})
	if err != nil {
		log.Fatalln("clickhouse.Open", err)
		return nil
	}
	return clickhouseConnect
}

func insertRequestsIfTime() {
	for {
		select { //задействует первый готовый канал
		case <-chInsert: //этот канал будет если была отправка по причине заполнения массива
			continue
		case <-time.After(time.Duration(AppConfig.RequestsTimeToSend) * time.Second): //этот канал будет готов по истечении времени
			//таким образом, отсчитывается время с момента последней отправки
			//log.Println("l_clickhouse.go_1")
			mtx.Lock()
			//log.Println("l_clickhouse.go_2")
			if len(toClickHouse) > 0 {
				dup := make([]proxyRow, len(toClickHouse))
				copy(dup, toClickHouse)
				toClickHouse = nil
				go reliableInsert(dup)
			}
			//log.Println("ul_clickhouse.go")
			mtx.Unlock()
		}
	}
}

func reliableInsert(arr []proxyRow) {
	//log.Println("l_clickhouse.go_1")
	mtx.Lock()
	//log.Println("l_clickhouse.go_2")
	inserts++
	insertsNow := inserts
	//log.Println("ul_clickhouse.go")
	mtx.Unlock()
	defer func() {
		//log.Println("l_clickhouse.go_1")
		mtx.Lock()
		//log.Println("l_clickhouse.go_2")
		inserts--
		//log.Println("ul_clickhouse.go")
		mtx.Unlock()
	}()
	var n = time.Now()
	for i := 1; i <= AppConfig.InsertAttemptsRequests; i++ {
		fmt.Printf("[%d] запись(1) %d строк, попытка %d\n", insertsNow, len(arr), i)
		if clickhouseInsert(arr) {
			fmt.Printf("[%d] запись(1) %d строк, попытка %d итого %vs ok\n", insertsNow, len(arr), i, time.Since(n).Seconds())
			break
		}
		time.Sleep(time.Duration(insertSleep) * time.Second)
	}
}

func clickhouseInsert(arr []proxyRow) bool {
	var conn = createConnectClickhouse()
	defer conn.Close()
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO stats_proxy (t, proxy, country, ip_output, latency, max_latency, good, PackId)")
	if err != nil {
		log.Println("conn.PrepareBatch", err)
		return false
	}
	chInsert <- true //отсчет времени в insertRequestsIfTime1() начнется с начала
	for i := 0; i < len(arr); i++ {
		r := arr[i]
		err = batch.Append(
			r.Datetime,
			r.proxy,
			r.Country,
			r.IpOut,
			r.Latency,
			r.MaxLatency,
			r.Good,
			r.PackId,
		)
		if err != nil {
			log.Println("batch.Append", err)
			log.Println("json", arr[i])
			continue // пропускаем проблемную строку
		}
	}
	if err := batch.Send(); err != nil {
		fmt.Println("batch.Send", err)
		return false
	}
	return true
}

func selectStatsForProxy(id int16) []fullProxyInfo {
	var r fullProxyInfo
	var ret []fullProxyInfo
	ret = make([]fullProxyInfo, 0)
	var conn = createConnectClickhouse()
	defer conn.Close()
	var query string
	var add1 string
	if id != -1 {
		add1 = fmt.Sprintf(" and sp.PackId = %d", id)
	}
	query = `WITH 
	    recent_stats AS (
	        SELECT
	        	sp.t,
	            sp.proxy,
	            sp.country,
	            sp.ip_output,
	            sp.latency,
	            sp.max_latency,
	            sp.good,
	            sp.PackId
	        FROM 
	            stats_proxy sp 
	        WHERE 
	            sp.t > now() - INTERVAL 1 HOUR`
	query += add1
	query += `),
	    max_t AS (
	        SELECT 
	            proxy, 
	            MAX(t) AS t1 
	        FROM 
	            recent_stats 
	        GROUP BY 
	            proxy
	    ),
	    aggregated_stats AS (
	        SELECT 
	            proxy, 
	            groupArray(latency) AS hist, 
	            groupArray(good) AS gd, 
	            varSamp(latency / 1e3) AS disp 
	        FROM 
	            recent_stats 
	        GROUP BY 
	            proxy
	    )
	SELECT 
	    sp.*, 
	    T2.hist, 
	    T2.gd, 
	    IF(isNaN(T2.disp), 0, T2.disp) AS disp
	FROM 
	    recent_stats sp
	JOIN 
	    max_t T 
	ON 
	    T.proxy = sp.proxy AND T.t1 = sp.t 
	JOIN 
	    aggregated_stats T2 
	ON 
	    sp.proxy = T2.proxy	
	ORDER BY 
	    T2.disp DESC;`
	ctx := context.Background()
	var rows, err = conn.Query(ctx, query)
	if err != nil {
		fmt.Println("conn.Query", err)
		return ret
	}
	for rows.Next() {
		err = rows.Scan(&r.Datatime, &r.Proxy, &r.Country, &r.IpOutput, &r.latency, &r.MaxLatency, &r.good, &r.packId, &r.HistLatency, &r.HistGood, &r.Disp)
		if err != nil {
			fmt.Println("selectStatsForProxy rows.Scan", err)
		}
		ret = append(ret, r)
	}
	return ret
}

func getLastHoursNotWork() []proxy {
	var ret = make([]proxy, 0)
	var query = "SELECT proxy\nFROM (\n    SELECT proxy, length(groupArray(good)) AS hist_length,\n           arrayExists(x -> x = TRUE, groupArray(good)) AS has_true\n    FROM stats_proxy\n    WHERE t > now() - INTERVAL 1 HOUR\n    GROUP BY proxy\n)\nWHERE hist_length > 5 AND not has_true"
	var conn = createConnectClickhouse()
	defer conn.Close()
	ctx := context.Background()
	rows, err := conn.Query(ctx, query)
	if err != nil {
		log.Println("getLastHoursNotWork: ", err)
		return nil
	}
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			fmt.Println("getLastHoursNotWork rows.Scan", err)
		}
		var p, _ = string2Proxy(s)
		ret = append(ret, p)
	}
	return ret
}

func selectWorkProxy() []statsWotk {
	var conn = createConnectClickhouse()
	var ret = make([]statsWotk, 0)
	var query = "WITH \n    recent_stats AS (\n        SELECT\n            proxy,\n            round(arrayAvg(groupArray(latency))) AS latency,\n            length(groupArray(ip_output)) AS hist_length,\n            arrayCount(x -> x != 0, groupArray(ip_output)) AS goods\n        FROM \n            stats_proxy      \n        WHERE \n        \tt > now() - INTERVAL 15 MINUTE  \n        GROUP BY \n            proxy\n    )\t\t\t\nSELECT \n    rs.proxy, \n    rs.goods * 100 / CAST(rs.hist_length AS Float32) AS percent,\n    rs.hist_length,\n    rs.latency\nFROM \n    recent_stats rs\nWHERE \n\tpercent >= 70 AND hist_length > 1 AND latency <= 7 * 1e6 ORDER BY hist_length DESC, percent DESC, latency ASC"
	ctx := context.Background()
	var rows, err = conn.Query(ctx, query)
	if err != nil {
		fmt.Println("conn.Query", err)
		return ret
	}
	for rows.Next() {
		r := statsWotk{}
		err = rows.Scan(&r.Proxy, &r.Percent, &r.LenHistory, &r.AvgLatency)
		if err != nil {
			fmt.Println("selectWorkProxy rows.Scan", err)
		}
		ret = append(ret, r)
	}
	return ret
}
