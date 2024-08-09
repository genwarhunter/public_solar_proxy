package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
)

var mySqlDB *sql.DB

type proxyDB struct {
	C string `json:"countW"`
	D string `json:"dateTime"`
}

type WorkProxyDB struct {
	Cw int    `json:"countW"`
	Ct int    `json:"countT"`
	D  string `json:"dateTime"`
}

func createConnectMysql() bool {
	var err error
	mySqlDB, err = sql.Open("mysql", AppConfig.DataSourceName)
	if err != nil {
		log.Fatalln("sql.Open", err)
		return false
	}
	return true
}

func selectFromPackage() ([]proxyPackageRow, bool) {
	var ans []proxyPackageRow
	var results *sql.Rows
	results, err := mySqlDB.Query("SELECT * FROM package")
	if err != nil {
		log.Println("selectFromPackage", err)
		return []proxyPackageRow{}, false
	}
	for results.Next() {
		var q proxyPackageRow
		err := results.Scan(&q.Id, &q.Name, &q.Url, &q.Load, &q.Use, &q.Weight)
		if err != nil {
			log.Println("selectFromPackage", err)
			return []proxyPackageRow{}, false
		}
		ans = append(ans, q)
	}
	_ = results.Close()
	return ans, true
}

func updateProxyPackage(domain string) {
	link := "http://91.210.107.92:10008" + "/parsers/" + domain + ".txt"
	var results *sql.Rows
	results, err := mySqlDB.Query("SELECT name FROM package WHERE name=?", domain)
	if err != nil {
		log.Println("updateProxyPackage", err)
		return
	}
	if results.Next() {
		return
	}

	_, err = mySqlDB.Query("INSERT INTO package (`name`, `url`, `load`, `use`, weight) values (?, ?, 1, 1, 100);", domain, link)
	if err != nil {
		log.Println("updateProxyPackage", err)
	}
}

func insertProxy(listProxy []proxy) {
	var placeholderLimit = 65535 // MySQL имеет ограничение в 65535 плейсхолдеров в подготовленном выражении
	//log.Println("l_sql.go")
	mtx.RLock()
	for i := 0; i < len(listProxy); i += placeholderLimit / 2 { // делим на 2, потому что каждая строка имеет 2 плейсхолдера
		end := i + placeholderLimit/2
		if end > len(listProxy) {
			end = len(listProxy)
		}
		subCluster := listProxy[i:end]
		var vals []interface{}
		var sqlStr = "INSERT INTO stats(proxy, PackId) VALUES "

		for _, p := range subCluster {
			sqlStr += "(?, ?),"
			var strProxy = proxy2String(p)
			vals = append(vals, strProxy, loaderUniq[p].mainPackageId)
		}
		sqlStr = sqlStr[:len(sqlStr)-1] // Удаляем последнюю запятую
		sqlStr += " ON DUPLICATE KEY UPDATE proxy=VALUES(proxy), packId=VALUES(packId)"

		// Начинаем транзакцию
		tx, err := mySqlDB.Begin()
		if err != nil {
			log.Println("insertProxy: ошибка при начале транзакции", err)
			return
		}

		// Обеспечиваем откат транзакции в случае ошибки
		defer tx.Rollback()

		stmt, err := tx.Prepare(sqlStr)
		if err != nil {
			log.Println("insertProxy: ошибка при подготовке выражения", err)
			return
		}

		_, err = stmt.Exec(vals...)
		if err != nil {
			log.Println("insertProxy: ошибка при выполнении выражения", err)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Println("insertProxy: ошибка при фиксации транзакции", err)
			return
		}

		stmt.Close()
	}
	//log.Println("ul_sql.go")
	mtx.RUnlock()

	log.Printf("Обновление %d прокси в базе данных", len(listProxy))
}

func selectFromStats() []proxyDB {
	var a proxyDB
	var ans = make([]proxyDB, 0)
	var results *sql.Rows
	results, err := mySqlDB.Query("SELECT COUNT(*) c, DATE_FORMAT(t0, \"%Y-%c-%d-%H-%i\") d from stats group by d ORDER by d ASC;")
	if err != nil {
		log.Println("selectFromCountry", err)
		return ans
	}
	for results.Next() {
		err := results.Scan(&a.C, &a.D)
		if err != nil {
			log.Println("selectFromStats", err)
			continue
		}
		ans = append(ans, a)
	}
	return ans
}

func selectCountWorkProxy() []WorkProxyDB {
	var a WorkProxyDB
	var ans = make([]WorkProxyDB, 0)
	var results *sql.Rows
	results, err := mySqlDB.Query("SELECT countWork, countTotal, DATE_FORMAT(t0, \"%Y-%c-%d-%H-%i\") d from workProxys group by d ORDER by d ASC;")
	if err != nil {
		log.Println("selectWorkProxy", err)
		return ans
	}
	for results.Next() {
		err := results.Scan(&a.Cw, &a.Ct, &a.D)
		if err != nil {
			log.Println("selectWorkProxy", err)
			continue
		}
		ans = append(ans, a)
	}
	err = results.Close()
	if err != nil {
		return nil
	}
	return ans
}

func selectDontCheckProxy() ([]proxy, []proxyInfo, []uint16) {
	var pArr = make([]proxy, 0)
	var infoArr = make([]proxyInfo, 0)
	var packs = make([]uint16, 0)
	results, err := mySqlDB.Query("select * from DontCheck")
	if err != nil {
		log.Println("selectWorkProxy", err)
		return pArr, infoArr, packs
	}
	for results.Next() {
		var pstring string
		var pkgId uint16
		err = results.Scan(&pstring, &pkgId)
		var p, info = string2Proxy(pstring)
		pArr = append(pArr, p)
		infoArr = append(infoArr, info)
		packs = append(packs, pkgId)
	}
	return pArr, infoArr, packs
}

func insertDontCheck() {
	var chunkSize = 10000
	var keys = make([]proxy, 0)
	mtx.RLock()
	for p, _ := range dontcheck {
		keys = append(keys, p)
	}
	l := len(keys)
	var n = l/chunkSize + 1
	mtx.RUnlock()
	for i := 0; i < n; i++ {
		var vals []interface{}
		minInd := i * chunkSize
		maxInd := (i + 1) * chunkSize
		if maxInd > l {
			maxInd = l
		}
		var cluster = keys[minInd:maxInd]
		var sqlStr = "INSERT INTO DontCheck (proxy, PackId) VALUES "
		mtx.RLock()
		for _, p := range cluster {
			sqlStr += "(?, ?),"
			var strProxy = proxy2String(p)
			var mainPackageId uint16
			v, ok := loaderUniq[p]
			if !ok {
				mainPackageId = v.mainPackageId
			} else {
				mainPackageId = 0
			}
			vals = append(vals, strProxy, mainPackageId)
		}
		mtx.RUnlock()
		sqlStr = sqlStr[:len(sqlStr)-1] // Remove the last comma
		sqlStr += " ON DUPLICATE KEY UPDATE proxy=VALUES(proxy), PackId=VALUES(PackId)"

		// Start the transaction
		tx, err := mySqlDB.Begin()
		if err != nil {
			log.Println("insertDontCheck: error beginning transaction", err)
			return
		}

		// Ensure the transaction is rolled back in case of an error
		defer tx.Rollback()

		stmt, err := tx.Prepare(sqlStr)
		if err != nil {
			log.Println("insertDontCheck: error preparing statement", err)
			return
		}

		_, err = stmt.Exec(vals...)
		if err != nil {
			log.Println("insertDontCheck: error executing statement", err)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Println("insertDontCheck: error committing transaction", err)
			return
		}

		stmt.Close()
	}
}

func removeFromDontCheck() {
	var sproxy = make([]any, 0)
	for _, p := range toRemove {
		sproxy = append(sproxy, proxy2String(p))
	}
	placeholders := strings.Repeat("?,", len(sproxy))
	if len(placeholders) == 0 {
		return
	}
	query := fmt.Sprintf("delete from DontCheck where proxy in (%s)", placeholders[:len(placeholders)-1])
	_, err := mySqlDB.Query(query, sproxy...)
	if err != nil {
		log.Println("removeFromDontCheck: ", err)
	}
	toRemove = make([]proxy, 0)
	return
}

func selectAllProxy() ([]proxy, []proxyInfo, []uint16) {
	var pArr = make([]proxy, 0)
	var infoArr = make([]proxyInfo, 0)
	var packArr = make([]uint16, 0)
	results, err := mySqlDB.Query("select proxy, packId from stats")
	if err != nil {
		log.Println("selectWorkProxy", err)
		return pArr, infoArr, nil
	}
	for results.Next() {
		var pstring string
		var packId int
		err = results.Scan(&pstring, &packId)
		var p, info = string2Proxy(pstring)
		pArr = append(pArr, p)
		infoArr = append(infoArr, info)
		packArr = append(packArr, uint16(packId))
	}
	return pArr, infoArr, packArr
}
