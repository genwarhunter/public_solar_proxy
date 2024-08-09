package main

import (
	"net/http"
	"os"
)

// генерирует и возвращает html страницу на основе /api/v2/stats
func statisticHandler(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/statistic.html")
	w.Write(ret)
}

// генерирует и возвращает html страницу на основе /api/v2/statsPackage
func statsPackageHandler(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/statsPackage.html")
	w.Write(ret)
}

func renderGraphsStat(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/graphsStat.html")
	w.Write(ret)
}

func renderGraphPerformance(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/graphPerformanse.html")
	w.Write(ret)
}

func getWorkProxylist(w http.ResponseWriter, r *http.Request) {

}

func settings(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/settings.html")
	w.Write(ret)
}

func instantCheckList(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/checker.html")
	w.Write(ret)
}

func getResultWebChecker(w http.ResponseWriter, r *http.Request) {
	ret, _ := os.ReadFile("./static/html/resultWebChecker.html")
	w.Write(ret)
}
