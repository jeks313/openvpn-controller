package main

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// @Title
// @Description
// @Author
// @Update

// Checker interface for the right side of the UI
type Checker interface {
	Check() error
	Status() CheckUI
}

// CheckUI has the elements needed for the HTML template
type CheckUI struct {
	Icon   string
	Status string
}

// CheckPing is host you would like to ping while VPN is up
type CheckPing struct {
	Host   string
	Active bool
	Stats  *probing.Statistics
}

// NewPing is host you would like to ping while VPN is up
func NewPing(host string) *CheckPing {
	return &CheckPing{Host: host}
}

// Status gets the status string/ui element
func (p *CheckPing) Status() CheckUI {
	var ui CheckUI
	if !p.Active || p.Stats == nil {
		ui.Status = fmt.Sprintf("Ping for %s is unknown", p.Host)
		ui.Icon = "fa-times-square"
		return ui
	}
	ui.Status = fmt.Sprintf("Ping for %s is %v", p.Host, p.Stats.AvgRtt)
	ui.Icon = "fa-check-square"
	return ui
}

// Check does the check
func (p *CheckPing) Check() error {
	pinger, err := probing.NewPinger(p.Host)
	if err != nil {
		slog.Error("failed to setup ping", "error", err)
		return err
	}
	pinger.Timeout = 1 * time.Second
	pinger.Count = 2
	err = pinger.Run()
	if err != nil {
		slog.Error("ping failed", "error", err)
		return err
	}
	p.Stats = pinger.Statistics()
	slog.Info("ping check", "host", p.Host, "ping", p.Stats.AvgRtt)
	p.Active = true
	return nil
}

// CheckDNS New host you would like to do DNS checks on when VPN is up
type CheckDNS struct {
	Host    string
	Address string
	Active  bool
}

// Status gets the status string/ui element
func (p *CheckDNS) Status() CheckUI {
	var ui CheckUI
	ui.Status = fmt.Sprintf("DNS for %s is %s", p.Host, p.Address)
	ui.Icon = "fa-check-square"
	if !p.Active {
		ui.Status = fmt.Sprintf("DNS for %s is unknown", p.Host)
		ui.Icon = "fa-times-square"
	}
	return ui
}

// Check does the check
func (p *CheckDNS) Check() error {
	ips, err := net.LookupIP(p.Host)
	p.Active = true
	if err != nil {
		p.Active = false
		return err
	}
	p.Address = fmt.Sprintf("%v", ips)
	slog.Info("dns check", "host", p.Host, "address", p.Address)
	return nil
}

// NewDNS host you would like to run DNS checks on when VPN is up
func NewDNS(host string) *CheckDNS {
	return &CheckDNS{Host: host}
}
