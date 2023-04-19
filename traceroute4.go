package traceroute

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
	"net/netip"
	"time"
)

func trace4(conf *TraceConfig, addr netip.Addr) ([]TraceResult, error) {
	if !addr.Is4() {
		return nil, fmt.Errorf("invalid addr:%s", addr.String())
	}
	if err := conf.check(); err != nil {
		return nil, err
	}

	sendSocket, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}
	defer unix.Close(sendSocket)
	recvSocket, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_ICMP)
	if err != nil {
		return nil, err
	}
	if err := unix.SetsockoptTimeval(recvSocket, unix.SOL_SOCKET, unix.SO_RCVTIMEO,
		&unix.Timeval{Sec: conf.WaitSec, Usec: 0}); err != nil {
		return nil, err
	}
	defer unix.Close(recvSocket)

	ttl := conf.FirstTTL
	try := conf.Retry
	destPort := DesMinPort
	destAddr := addr.As4()

	var results []TraceResult
	begin := time.Now()
	for {
		begin = time.Now()

		if err := unix.SetsockoptInt(sendSocket, 0x0, unix.IP_TTL, ttl); err != nil {
			return nil, err
		}
		if err := unix.Sendto(sendSocket, []byte{0}, 0, &unix.SockaddrInet4{Port: destPort, Addr: destAddr}); err != nil {
			return nil, err
		}

		var p = make([]byte, 4096)
		result := TraceResult{TTL: ttl, ElapsedTime: time.Since(begin), Replied: false}
		n, from, err := unix.Recvfrom(recvSocket, p, 0)
		if err == nil {
			try = 0
			fromAddr := from.(*unix.SockaddrInet4).Addr
			ipHeader, err := ipv4.ParseHeader(p[:n])
			if err != nil {
				return nil, err
			}
			if ipHeader.Len > n {
				continue
			}

			icmpReply, err := icmp.ParseMessage(1, p[ipHeader.Len:n])
			if err != nil {
				return nil, err
			}
			if icmpReply.Type == ipv4.ICMPTypeTimeExceeded || icmpReply.Type == ipv4.ICMPTypeDestinationUnreachable {
				result.Replied = true
				result.NextHot = netip.AddrFrom4(fromAddr).String()
				results = append(results, result)
				if conf.Debug {
					fmt.Printf("ttl %d addr:%v time:%v \n", ttl, result.NextHot, time.Since(begin))
				}
				if icmpReply.Type == ipv4.ICMPTypeTimeExceeded {
					ttl++
				}
				if icmpReply.Type == ipv4.ICMPTypeDestinationUnreachable || ttl > conf.MaxTTL || fromAddr == destAddr {
					return results, nil
				}
			} else {
				fmt.Printf("%d unknown:%+v from:%+v\n", ttl, icmpReply, fromAddr)
			}
		} else {
			if conf.Debug {
				fmt.Printf("ttl %d * err:%s time:%v\n", ttl, err.Error(), time.Since(begin))
			}
			result.NextHot = "*"
			results = append(results, result)
			try++
			if try > conf.Retry {
				try = 0
				ttl++
			}
			if ttl > conf.MaxTTL {
				return results, nil
			}
		}

		destPort++
		if destPort > DesMaxPort {
			destPort = DesMinPort
		}
	}
}
