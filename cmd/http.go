package main

import (
	"html/template"
	"net/http"
)

// @Title
// @Description
// @Author
// @Update

func PostCredentials(w http.ResponseWriter, r *http.Request) {
}

func GetIndex(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
	t.Execute(w, nil)
}

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
