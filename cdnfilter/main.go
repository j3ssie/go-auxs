package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

// Most of the file literally copied from my friend @thebl4ckturtle code

var ReservedCIDRs = []string{
	"192.168.0.0/16",
	"172.16.0.0/12",
	"10.0.0.0/8",
	"127.0.0.0/8",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"100.64.0.0/10",
	"198.18.0.0/15",
	"169.254.0.0/16",
	"192.88.99.0/24",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.94.77.0/24",
	"192.94.78.0/24",
	"192.52.193.0/24",
	"192.12.109.0/24",
	"192.31.196.0/24",
	"192.0.0.0/29",
}

// The reserved network address ranges
var reservedAddrRanges []*net.IPNet

func init() {
	for _, cidr := range ReservedCIDRs {
		if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
			reservedAddrRanges = append(reservedAddrRanges, ipnet)
		}
	}
}

func main() {
	var cdnOutputFile string
	var notCdnOutputFile string
	flag.StringVar(&cdnOutputFile, "c", "cdn.txt", "CDN output file")
	flag.StringVar(&notCdnOutputFile, "n", "non-cdn.txt", "None CDN output file")
	flag.Parse()
	if cdnOutputFile == "" || notCdnOutputFile == "" {
		fmt.Fprintf(os.Stderr, "Check your input again\n")
		os.Exit(1)
	}

	cdnOutput, err := os.OpenFile(cdnOutputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create/open cdnOutput\n")
		os.Exit(1)
	}
	defer cdnOutput.Close()

	notCdnOutput, err := os.OpenFile(notCdnOutputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create/open notCdnOutputFile\n")
		os.Exit(1)
	}
	defer notCdnOutput.Close()

	client, err := NewCDNCheck()
	if err != nil {
		log.Fatal(err)
	}
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		ip := net.ParseIP(line)
		if !isPrivateIP(ip) {
			continue
		}
		if ip == nil {
			continue
		}
		if line == "0.0.0.0" {
			continue
		}
		if localIP, _ := isReservedAddress(line); localIP {
			// fmt.Println("Reserved Address: ", localIP)
			continue
		}
		found, err := client.Check(ip)
		if err != nil {
			continue
		}
		if found {
			_, _ = cdnOutput.WriteString(line + ":80\n")
			_, _ = cdnOutput.WriteString(line + ":443\n")

		} else {
			fmt.Println(line)
			_, _ = notCdnOutput.WriteString(line + "\n")
			// print nonCDN ip out
		}
	}
}

// IsReservedAddress checks if the addr parameter is within one of the address ranges in the ReservedCIDRs slice.
func isReservedAddress(addr string) (bool, string) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false, ""
	}

	var cidr string
	for _, block := range reservedAddrRanges {
		if block.Contains(ip) {
			cidr = block.String()
			break
		}
	}

	if cidr != "" {
		return true, cidr
	}
	return false, ""
}

// Copying from https://github.com/audiolion/ipip
// isPrivateIP check if IP is private or not
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 || (ip4[0] == 172 && ip4[1]&0xf0 == 16) || (ip4[0] == 192 && ip4[1] == 168)
	}
	return len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc
}
