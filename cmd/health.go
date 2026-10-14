package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
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
	AvgRtt time.Duration
}

// NewPing is host you would like to ping while VPN is up
func NewPing(host string) *CheckPing {
	return &CheckPing{Host: host}
}

// Status gets the status string/ui element
func (p *CheckPing) Status() CheckUI {
	var ui CheckUI
	if !p.Active {
		ui.Status = fmt.Sprintf("Ping for %s is unknown", p.Host)
		ui.Icon = "fa-times-square"
		return ui
	}
	ui.Status = fmt.Sprintf("Ping for %s is %v", p.Host, p.AvgRtt)
	ui.Icon = "fa-check-square"
	return ui
}

// Check does the check: two ICMP echoes within a second, averaging the replies
func (p *CheckPing) Check() error {
	rtt, err := ping(p.Host, 2, 1*time.Second)
	if err != nil {
		slog.Error("ping failed", "host", p.Host, "error", err)
		p.Active = false
		return err
	}
	p.AvgRtt = rtt
	slog.Info("ping check", "host", p.Host, "ping", p.AvgRtt)
	p.Active = true
	return nil
}

// ping sends count ICMP echo requests to host and returns the average round
// trip of the replies that arrive before timeout. It uses an unprivileged
// datagram ICMP socket (net.ipv4.ping_group_range) when the kernel allows it,
// and a raw socket (CAP_NET_RAW) otherwise.
func ping(host string, count int, timeout time.Duration) (time.Duration, error) {
	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return 0, err
	}
	var peer net.Addr = &net.UDPAddr{IP: dst.IP}
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		peer = dst
		conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			return 0, err
		}
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return 0, err
	}

	// The kernel rewrites the ID on datagram sockets, so replies are matched
	// on sequence number; the ID only filters other pingers on a raw socket.
	id := os.Getpid() & 0xffff
	sent := make(map[int]time.Time, count)
	for seq := 1; seq <= count; seq++ {
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Body: &icmp.Echo{ID: id, Seq: seq, Data: []byte("openvpn-controller")},
		}
		b, err := msg.Marshal(nil)
		if err != nil {
			return 0, err
		}
		if _, err := conn.WriteTo(b, peer); err != nil {
			return 0, err
		}
		sent[seq] = time.Now()
	}

	var total time.Duration
	received := 0
	buf := make([]byte, 1500)
	for received < count {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			return 0, err
		}
		reply, err := icmp.ParseMessage(1, buf[:n]) // 1 = ICMPv4
		if err != nil || reply.Type != ipv4.ICMPTypeEchoReply {
			continue
		}
		echo, ok := reply.Body.(*icmp.Echo)
		if !ok {
			continue
		}
		if _, raw := peer.(*net.IPAddr); raw && echo.ID != id {
			continue
		}
		start, ok := sent[echo.Seq]
		if !ok {
			continue
		}
		delete(sent, echo.Seq)
		total += time.Since(start)
		received++
	}
	if received == 0 {
		return 0, fmt.Errorf("no echo reply from %s within %v", host, timeout)
	}
	return total / time.Duration(received), nil
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
