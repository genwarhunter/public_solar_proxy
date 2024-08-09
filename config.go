package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// для тестирования
const TEST_RANDOM_PROXY = false                 // true - вместо загрузки реальных списков прокси сгенерировать случайные
const TEST_RANDOM_PROXIES_PER_PKG = 6000        // если TEST_RANDOM_PROXY сколько проксей генерировать для каждого пакета
const TEST_CHECKER_ALWAYS_OK = false            // true - чекер без проверки сразу присваивает ipOutput == proxy.ip
const TEST_CHECKER_SLEEP = 1 * time.Millisecond // если TEST_CHECKER_ALWAYS_OK задержка работы чекера

const REFRESH_SLEEP = 60 * time.Second // периодичность получения всех таблиц из базы (пакеты, юзерагенты, страны)

var AppConfig Conf

type Conf struct {
	RequestsTimeToSend        int    `json:"requestsTimeToSend"`        // Количество секунд, через сколько делать запрос на insert в clickhhose
	InsertAttemptsRequests    int    `json:"insertAttemptsRequests"`    // Количество попыток insert в clickhhose
	HistSecPerformanceThreads int    `json:"histSecPerformanceThreads"` // Секунды, длительность сохранения истории потоков и очереди
	SaveProxyDB               int    `json:"saveProxyDB"`               // Минуты, периодичность сохранения всех прокси в бд
	CheckerTimeoutSeconds     int    `json:"checkerTimeoutSeconds"`     // Секунды, таймаут запросов чекера
	CheckerStreamsMin         int    `json:"checkerStreamsMin"`         // Минимальное число потоков чекера
	CheckerStreamsMax         int    `json:"checkerStreamsMax"`         // Максимальное число потоков чекера
	CheckerURL                string `json:"checkerURL"`                // URL адрес на который будет делать запрос чекер, если пусто - попытаться определить внешний ip адрес текущего сервера и делать запросы на него
	LoaderPeriodMinutes       int    `json:"loaderPeriodMinutes"`       // Минуты, Периодичность загрузки списка прокси для каждого пакета
	CheckerPackagesSecond     int    `json:"checkerPackagesSecond"`     // Секунды, через какое время делать перераспределение прокси по пакетам
	CheckerVerbose            bool   // Подробный вывод результатов работы чекера по каждой прокси, только для отладки
	ClickhouseAddr            string // ip:port clickhouse
	ClickhouseDB              string // Database
	ClickhouseUser            string // User
	ClickhousePassword        string // password
	ListenAddr                string // IP:Port для прослушивания
	CheckerInsecure           bool   // Отключить проверку сертификата для запросов чекером
	DataSourceName            string // Mysql
}

func getConfig() {
	file, err := os.Open("conf.json")
	defer file.Close()
	if err != nil {
		log.Panicln("Error occurred while reading config")
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&AppConfig)
	if err != nil {
		log.Panicln("Invalid json")
	}
}
