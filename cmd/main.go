package main

import (
	"flag"
	"fmt"

	"github.com/Kseleven/traceroute-go"
)

func main() {
	conf := &traceroute.TraceConfig{
		Debug: true,
	}

	var destAddr string
	flag.IntVar(&conf.FirstTTL, "f", 1, "first ttl")
	flag.IntVar(&conf.MaxTTL, "m", 30, "max ttl")
	flag.IntVar(&conf.Retry, "r", 0, "retry time")
	flag.Int64Var(&conf.WaitSec, "w", 1, "wait seconds")

	flag.Parse()
	destAddr = flag.Arg(0)
	if destAddr == "" {
		usage()
		return
	}

	fmt.Printf("traceroute to %s %d hots max\n", destAddr, conf.MaxTTL)
	results, err := traceroute.Traceroute(destAddr, conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("results:%+v\n", results)
}

func usage() {
	fmt.Println("usage: traceroute host(dest address ipv4 or ipv6)")
	fmt.Println("with config:traceroute [-f firstTTL] [-m maxTTL] [-r retryTimes] [-w wait seconds] host(ipv4 or ipv6)]")
}
