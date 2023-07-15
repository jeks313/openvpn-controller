package main

import (
	"fmt"
	"html/template"
	"net/http"
)

// @Title
// @Description
// @Author
// @Update

// PostConnect handles a connect post request from the UI, inputs are username, password and one-time-code
func PostConnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Write([]byte("testing!"))

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	otp := r.PostFormValue("one-time-code")

	w.Write([]byte(fmt.Sprintf("%s/%s/%s", username, password, otp)))
}

// GetIndex serves up the index page
func GetIndex(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
	t.Execute(w, nil)
}

// GetVPNStatus is currently a test thingumy
func GetVPNStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Write([]byte("testing!"))
}

/*
// HistoryQueryHandler will dump the ring buffer of historical slow queries
func HistoryQueryHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		var queries []*Query
		slow.history.Do(func(p interface{}) {
			if p != nil {
				queries = append(queries, p.(*Query))
			}
		})
		json.NewEncoder(w).Encode(queries)
	}
}

// RunningQueryTableHandler will output the running queries in a datatable
func RunningQueryTableHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("table").Parse(queriesHTML))
	return func(w http.ResponseWriter, r *http.Request) {
		var queries []*Query
		for _, query := range slow.runningQueries {
			queries = append(queries, query)
		}
		w.Header().Set("content-type", "text/html")
		j, _ := json.Marshal(queries)
		t.Execute(w, string(j))
	}
}

// HistoryQueryTableHandler will output the running queries in a datatable
func HistoryQueryTableHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("table").Parse(queriesHTML))
	return func(w http.ResponseWriter, r *http.Request) {
		var queries []*Query
		slow.history.Do(func(p interface{}) {
			if p != nil {
				queries = append(queries, p.(*Query))
			}
		})
		w.Header().Set("content-type", "text/html")
		j, _ := json.Marshal(queries)
		t.Execute(w, string(j))
	}
}
*/
