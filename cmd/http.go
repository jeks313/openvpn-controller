package main

import (
	"bytes"
	"container/ring"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// @Title
// @Description
// @Author
// @Update

// PostConnect handles a connect post request from the UI, inputs are username, password and one-time-code
func PostConnect(o *OpenVPN) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		otp := r.PostFormValue("one-time-code")

		slog.Info("received connect request", "username", username)

		c := Credentials{
			Username:    username,
			Password:    password,
			OneTimeCode: otp,
		}

		// reset the credentials form
		t := template.Must(template.ParseFS(templateFS, "templates/openvpn.html"))
		t.ExecuteTemplate(w, "credentials", nil)

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

		// update connected panel
		t.ExecuteTemplate(w, "connected", "yes! I'm connecting now ...")
	}
}

// GetLog gets the log data from a log history circular buffer
func GetLog(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Join(history.Lines(), "\n")))
	}
}

// GetUpdateWs
func GetUpdateWs(d *Display, history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		now := d.Updates
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
			w, err := ws.NextWriter(websocket.TextMessage)
			if err != nil {
				slog.Error("failed to get writer to websocket", "error", err)
				return
			}
			d.Log(w, history)
			for {
				if now == d.Updates {
					break
				}
				w, err := ws.NextWriter(websocket.TextMessage)
				if err != nil {
					slog.Error("failed to get writer to websocket", "error", err)
					return
				}
				slog.Debug("writing pending ui updates")
				w.Write(now.Value.([]byte))
				now = now.Next()
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// Display takes care of all HTML display tasks
type Display struct {
	TemplateFile string
	Template     *template.Template
	Updates      *ring.Ring
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
	d.Updates = ring.New(5)
	return d, err
}

type VPNUI struct {
	PID    string
	Status string
}

func (d *Display) VPN(w io.Writer, o *OpenVPN) {
	v := VPNUI{
		PID:    "something or other",
		Status: "kind of running",
	}
	if o.running && o.cmd != nil {
		v.PID = fmt.Sprintf("%v", o.cmd.Process.Pid)
		v.Status = "OpenVPN process is running!"
	}
	d.ExecuteTemplateUpdate(w, "vpn", v)
}

func GetVPN(d *Display, o *OpenVPN) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d.VPN(w, o)
	}
}

func (d *Display) Index(w io.Writer, checks []Checker) {
	var c []CheckUI
	for _, i := range checks {
		c = append(c, i.Status())
	}
	d.Template.Execute(w, nil)
}

type LogUI struct {
	Line string
}

func (d *Display) Log(w io.Writer, history *LogHistory) {
	var h []LogUI
	lines := history.Lines()
	for _, line := range lines {
		h = append(h, LogUI{Line: line})
	}
	m := map[string][]LogUI{"Log": h}
	d.Template.ExecuteTemplate(w, "log", m)
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

func (d *Display) ExecuteTemplateUpdate(w io.Writer, fragment string, data any) {
	var buf bytes.Buffer
	mw := io.MultiWriter(&buf, w)
	d.Template.ExecuteTemplate(mw, fragment, data)
	d.Updates.Value = buf.Bytes()
	d.Updates = d.Updates.Next()
}

func (d *Display) VPNStatus(w io.Writer, checks []Checker) {
	var c []CheckUI
	for _, i := range checks {
		c = append(c, i.Status())
	}
	m := map[string][]CheckUI{
		"Checks": c,
	}
	d.ExecuteTemplateUpdate(w, "check-list-item", m)
}

// GetVPNStatus is currently a test thingumy
func GetVPNStatus(d *Display, checks []Checker) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d.VPNStatus(w, checks)
	}
}

type NullWriter struct{}

func (nw *NullWriter) Write(data []byte) (n int, err error) {
	return len(data), nil
}
