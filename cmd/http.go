package main

import (
	"container/ring"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"
)

// @Title
// @Description
// @Author
// @Update

func GetVPN(o *OpenVPN) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if o.running && o.cmd != nil {
			w.Write([]byte("I'm running"))
			w.Write([]byte(fmt.Sprintf("%v", o.cmd.Process.Pid)))
		}
	}
}

// PostConnect handles a connect post request from the UI, inputs are username, password and one-time-code
func PostConnect(o *OpenVPN) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		otp := r.PostFormValue("one-time-code")

		slog.Info("received connect request", "username", username)

		// reset the credentials form
		t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
		t.ExecuteTemplate(w, "credentials", nil)

		c := Credentials{
			Username:    username,
			Password:    password,
			OneTimeCode: otp,
		}

		filename := "/tmp/chyde.conf"
		err := c.Store(filename)
		if err != nil {
			slog.Error("unable to store credentials", "error", err)
			return
		}
		slog.Info("wrote credentials file", "filename", filename)
		err = o.Start(c.OneTimeCode)
		if err != nil {
			slog.Error("failed to start openvpn client", "error", err)
			t.ExecuteTemplate(w, "connected", fmt.Sprintf("failed to connect %v", err))
			return
		}
		slog.Info("started vpn client")
		t.ExecuteTemplate(w, "connected", "yes! I'm connected now")
	}
}

// GetLog gets the log data from a log history circular buffer
func GetLog(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Write([]byte(history.String()))
	}
}

// GetIndex serves up the index page
func GetIndex(checks []Checker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
		var c []CheckUI
		for _, i := range checks {
			c = append(c, i.Status())
		}
		//m := map[string][]CheckUI{
		//	"Checks": c,
		//}
		t.Execute(w, nil)
	}
}

// GetLogStream streams the log to the endpoint
func GetLogStream(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var h *ring.Ring = history.History
		rc := http.NewResponseController(w)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		for {
			for {
				if h == history.History {
					break
				}
				w.Write([]byte(h.Value.(string)))
				h = h.Next()
			}
			rc.Flush()
			time.Sleep(time.Millisecond * 500)
		}
	}
}

// GetVPNStatus is currently a test thingumy
func GetVPNStatus(checks []Checker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
		var c []CheckUI
		for _, i := range checks {
			c = append(c, i.Status())
		}
		m := map[string][]CheckUI{
			"Checks": c,
		}
		t.ExecuteTemplate(w, "check-list-item", m)
	}
}

/*
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
