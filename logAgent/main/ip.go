package main

import (
	"fmt"
	"net"
	"os"

	"github.com/astaxie/beego/logs"
)

var (
	LocalIPs []string
)

func init() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logs.Error("init get addrs failed ,err:", err)
		// os.Stderr.WriteString("Oops:" + err.Error())
		// os.Exit(1)
		return
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// os.Stdout.WriteString(ipnet.IP.String() + "\n")
				LocalIPs = append(LocalIPs, ipnet.IP.String())
			}
		}
	}
	fmt.Println(LocalIPs)
}

//获取ipv4地址
func get_internal() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("Oops:" + err.Error())
		os.Exit(1)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				os.Stdout.WriteString(ipnet.IP.String() + "\n")

			}
		}
	}
	os.Exit(0)
}
