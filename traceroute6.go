package traceroute

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
	"net/netip"
	"time"
)

func trace6(conf *TraceConfig, addr netip.Addr) ([]TraceResult, error) {
	if !addr.Is6() {
		return nil, fmt.Errorf("invalid addr:%s", addr.String())
	}
	if err := conf.check(); err != nil {
		return nil, err
	}
	ttl := conf.FirstTTL
	try := conf.Retry
	destPort := DesMinPort
	destAddr := addr.As16()

	recvSocket, err := unix.Socket(unix.AF_INET6, unix.SOCK_RAW, unix.IPPROTO_ICMPV6)
	if err != nil {
		return nil, fmt.Errorf("socket recv int failed:%s", err.Error())
	}
	sendSocket, err := unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("socket send int failed:%s", err.Error())
	}
	defer unix.Close(recvSocket)
	defer unix.Close(sendSocket)

	if err := unix.SetsockoptTimeval(recvSocket, unix.SOL_SOCKET, unix.SO_RCVTIMEO,
		&unix.Timeval{Sec: conf.WaitSec, Usec: 0}); err != nil {
		return nil, fmt.Errorf("socket opt recv int failed:%s", err.Error())
	}

	var results []TraceResult
	begin := time.Now()
	for {
		begin = time.Now()
		if err := unix.SetsockoptInt(sendSocket, unix.IPPROTO_IPV6, unix.IPV6_UNICAST_HOPS, ttl); err != nil {
			return nil, fmt.Errorf("socket opt ttl int failed:%s", err.Error())
		}
		if err := unix.Sendto(sendSocket, []byte("hello"), 0, &unix.SockaddrInet6{Port: destPort, Addr: destAddr}); err != nil {
			return nil, fmt.Errorf("sendto failed:%s", err.Error())
		}

		var p = make([]byte, 4096)
		result := TraceResult{TTL: ttl, ElapsedTime: time.Since(begin), Replied: false}
		n, from, err := unix.Recvfrom(recvSocket, p, 0)
		if err == nil {
			try = 0
			icmpReply, err := icmp.ParseMessage(58, p[:n])
			if err != nil {
				return nil, fmt.Errorf("parse message failed:%s", err.Error())
			}

			if icmpReply.Type == ipv6.ICMPTypeTimeExceeded || icmpReply.Type == ipv6.ICMPTypeDestinationUnreachable {
				fromAddr := from.(*unix.SockaddrInet6).Addr
				result.Replied = true
				result.NextHot = netip.AddrFrom16(fromAddr).String()
				results = append(results, result)
				if conf.Debug {
					fmt.Printf("ttl %d receive from:%v time:%v icmpReply:%+v\n", ttl, result.NextHot, time.Since(begin), icmpReply)
				}

				if icmpReply.Type == ipv6.ICMPTypeTimeExceeded {
					ttl++
				}
				if icmpReply.Type == ipv6.ICMPTypeDestinationUnreachable || ttl > conf.MaxTTL || fromAddr == destAddr {
					return results, nil
				}
			}
		} else {
			if conf.Debug {
				fmt.Printf("ttl %d * err: %s \n", ttl, err.Error())
			}
			result.NextHot = "*"
			results = append(results, result)
			try++
			if try > conf.Retry {
				try = 0
				ttl++
			}
		}

		destPort++
		if destPort > DesMaxPort {
			destPort = DesMinPort
		}
	}
}
