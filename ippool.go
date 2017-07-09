package main

import (
	"errors"
	"net"
)

type ippool struct {
	pool []net.IP
	Net  *net.IPNet
}

func (p *ippool) Pop() (net.IP, error) {
	if len(p.pool) <= 0 {
		return nil, errors.New("no more ip to return")
	}
	ip := p.pool[0]
	p.pool = p.pool[1:]
	return ip, nil
}

func (p *ippool) Push(ip net.IP) {
	p.pool = append(p.pool, ip)
}

func (p *ippool) Lenght() int {
	return len(p.pool)
}

func newIPPool(cidr string) (*ippool, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	nb, _ := ipnet.Mask.Size()
	if nb == 32 {
		return nil, errors.New("no ip valid in this range")
	}
	if nb < 16 {
		return nil, errors.New("too many ips")
	}
	p := &ippool{}
	p.Net = ipnet
	nHosts := 2<<uint(31-nb) - 2
	intIP := ipToInt(ip.String())
	for i := 1; i <= nHosts; i++ {
		tIP := ipIntToIP(intIP + i)
		p.Push(tIP)
	}
	return p, nil
}
