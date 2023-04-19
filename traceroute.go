package traceroute

import (
	"fmt"
	"net/netip"
	"time"
)

type TraceResult struct {
	TTL         int
	NextHot     string
	ElapsedTime time.Duration
	Replied     bool
}

type TraceConfig struct {
	FirstTTL int
	Retry    int
	MaxTTL   int
	Debug    bool
	WaitSec  int64
}

func (conf *TraceConfig) check() error {
	if conf.MaxTTL <= 0 {
		return fmt.Errorf("invalid max ttl: %d", conf.MaxTTL)
	}

	if conf.FirstTTL <= 0 {
		conf.FirstTTL = DefaultFirstTTL
	}

	if conf.MaxTTL > DefaultMaxTTL {
		conf.MaxTTL = DefaultMaxTTL
	}

	if conf.WaitSec <= 0 {
		conf.WaitSec = DefaultMinWaitSec
	} else if conf.WaitSec >= DefaultMaxWaitSec {
		conf.WaitSec = DefaultMaxWaitSec
	}

	return nil
}

const (
	DesMinPort        = 33434
	DesMaxPort        = 33534
	DefaultFirstTTL   = 1
	DefaultMaxTTL     = 64
	DefaultMinWaitSec = 1
	DefaultMaxWaitSec = 10
)

func Traceroute(destIP string, conf *TraceConfig) ([]TraceResult, error) {
	addr, err := netip.ParseAddr(destIP)
	if err != nil {
		return nil, err
	}

	if conf == nil {
		conf = &TraceConfig{
			FirstTTL: 1,
			Retry:    0,
			MaxTTL:   30,
		}
	}
	if addr.Is4() {
		return trace4(conf, addr)
	} else {
		return trace6(conf, addr)
	}
}
