package main

// @Title
// @Description
// @Author
// @Update

// Statuser interface for the right side of the UI
type Statuser interface {
	Check() error
	Render() (string, error)
}

// Pinger is host you would like to ping while VPN is up
type Pinger struct {
	Host    string
	Latency int
}

// NewPinger is host you would like to ping while VPN is up
func NewPinger(host string) *Pinger {
	return &Pinger{Host: host}
}

// DNSer New host you would like to do DNS checks on when VPN is up
type DNSer struct {
	Host    string
	Address string
}

// NewDNSer host you would like to run DNS checks on when VPN is up
func NewDNSer(host string) *DNSer {
	return &DNSer{Host: host}
}
