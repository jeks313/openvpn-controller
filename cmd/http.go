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
		if err := t.ExecuteTemplate(w, "credentials", nil); err != nil {
			slog.Error("render credentials", "error", err)
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
			if err := t.ExecuteTemplate(w, "connected", fmt.Sprintf("failed to connect %v", err)); err != nil {
				slog.Error("render connected", "error", err)
			}
			return
		}
		slog.Info("started vpn client")
	}
}

// GetLog gets the log data from a log history circular buffer
func GetLog(history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Join(history.Lines(), "\n")))
	}
}

// GetUpdates streams UI fragments to the page as server-sent events: the log
// every second, plus any check/VPN fragments rendered since the last pass. The
// page swaps each top-level element in by its id.
func GetUpdates(d *Display, history *LogHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		now := d.Updates
		rc := http.NewResponseController(w)
		// The server's WriteTimeout would otherwise end the stream after 15s.
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			slog.Error("failed to clear write deadline for event stream", "error", err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		slog.Info("opened update stream")
		for {
			var buf bytes.Buffer
			d.Log(&buf, history)
			for now != d.Updates {
				slog.Debug("writing pending ui updates")
				buf.Write(now.Value.([]byte))
				now = now.Next()
			}
			writeEvent(w, buf.Bytes())
			if err := rc.Flush(); err != nil {
				slog.Error("failed to flush update stream", "error", err)
				return
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(1 * time.Second):
			}
		}
	}
}

// writeEvent writes data as one server-sent event; every line needs its own
// "data:" prefix.
func writeEvent(w io.Writer, data []byte) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = io.WriteString(w, "\n")
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
	if err := d.Template.Execute(w, nil); err != nil {
		slog.Error("render index", "error", err)
	}
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
	if err := d.Template.ExecuteTemplate(w, "log", m); err != nil {
		slog.Error("render log", "error", err)
	}
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
		h := history.History
		rc := http.NewResponseController(w)
		for {
			for h != history.History {
				_, _ = w.Write([]byte(h.Value.(string)))
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
	if err := d.Template.ExecuteTemplate(mw, fragment, data); err != nil {
		slog.Error("render update", "fragment", fragment, "error", err)
	}
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
