package main

import (
	"container/ring"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
		t.ExecuteTemplate(w, "connected", "yes! I'm connecting now ...")
	}
}

// GetLog gets the log data from a log history circular buffer
func GetLog(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(history.String()))
	}
}

// GetUpdateWs
func GetUpdateWs(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				slog.Error("failed to upgrade to websocket", "error", err)
			}
			slog.Error("failed to upgrade to websocket")
			return
		}
		slog.Info("upgraded websocket call")
		for {
			data := fmt.Sprintf("<div class=\"is-family-monospace is-size-7\" id=\"logdetail\">%s</div>",
				history.String())
			err = ws.WriteMessage(websocket.TextMessage, []byte(data))
			if err != nil {
				slog.Error("failed to write to websocket", "error", err)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// Display takes care of all HTML display tasks
type Display struct {
	TemplateFile string
	Template     *template.Template
}

func NewDisplay(templateFile string) (*Display, error) {
	d := &Display{
		TemplateFile: templateFile,
	}
	t, err := template.ParseFS(templateFS, d.TemplateFile)
	if err != nil {
		slog.Error("failed to parse template", "filename", templateFile)
	}
	d.Template = t
	return d, err
}

func (d *Display) Index(w io.Writer, checks []Checker) {
	var c []CheckUI
	for _, i := range checks {
		c = append(c, i.Status())
	}
	d.Template.Execute(w, nil)
}

// GetIndex serves up the index page
func GetIndex(d *Display, checks []Checker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d.Index(w, checks)
	}
}

// GetLogStream streams the log to the endpoint
func GetLogStream(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var h *ring.Ring = history.History
		rc := http.NewResponseController(w)
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

func (d *Display) VPNStatus(w io.Writer, checks []Checker) {
	var c []CheckUI
	for _, i := range checks {
		c = append(c, i.Status())
	}
	m := map[string][]CheckUI{
		"Checks": c,
	}
	d.Template.ExecuteTemplate(w, "check-list-item", m)
}

// GetVPNStatus is currently a test thingumy
func GetVPNStatus(d *Display, checks []Checker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d.VPNStatus(w, checks)
	}
}
