package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/c-robinson/iplib"
)

// Extend the IP range by CIDR
// Usage: echo '1.2.3.4/24' | eip -s 32
// Usage: echo '1.2.3.4/24' | eip -p small

var (
	unique bool
	pURL   bool
	sub    int
	port   string
	ports  []string
)

var (
	pSmall  []string
	pMedium []string
	pLarge  []string
)

func main() {
	pSmall = []string{"80", "443", "8000", "8080", "8081", "8443", "9000", "9200"}
	pMedium = append(pSmall, []string{"81", "3000", "6066", "6443", "8008", "8083", "8834", "8888", "9091", "9443"}...)
	pLarge = append(pMedium, []string{"591", "2082", "2087", "2095", "2096", "4444", "4040", "6066", "9092", "10250", "10251"}...)

	// cli arguments
	flag.BoolVar(&pURL, "U", true, "parse url pattern too (only affected with '-p' option)")
	flag.BoolVar(&unique, "u", true, "unique result")
	flag.IntVar(&sub, "s", 32, "CIDR subnet (e.g: 24, 22)")
	flag.StringVar(&port, "p", "", "Append port after each IP (some predefined value: full, xlarge, large,medium, small or f,x,l,m,s)")
	flag.Parse()

	if port != "" {
		ports = genPorts(port)
	}

	var result []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		job := strings.TrimSpace(sc.Text())
		data := extendRange(job, sub)
		if len(data) > 0 {
			result = append(result, data...)
		}

		if pURL {
			data := extendURL(job)
			if len(data) > 0 {
				result = append(result, data...)
			}
		}

	}

	if !unique {
		fmt.Println(strings.Join(result, "\n"))
		return
	}

	unique := make(map[string]bool)
	for _, v := range result {
		if !unique[v] {
			unique[v] = true
			fmt.Println(v)
		}
	}
}

func extendURL(raw string) []string {
	var result []string
	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" || strings.Contains(u.Scheme, ".") {
		raw = fmt.Sprintf("http://%v", raw)
		u, err = url.Parse(raw)
		if err != nil {
			return result
		}
	}

	for _, p := range ports {
		result = append(result, fmt.Sprintf("%s:%s", u.Hostname(), p))
	}
	return result
}

func extendRange(rangeIP string, sub int) []string {
	var result []string
	_, ipna, err := iplib.ParseCIDR(rangeIP)
	if err != nil {
		ip := net.ParseIP(rangeIP)
		if ip != nil {
			if port == "" || sub != 32 {
				fmt.Println(ip)
			} else {
				for _, p := range ports {
					fmt.Printf("%s:%s\n", ip, p)
				}
			}
		}
		return result
	}
	extendedIPs, err := ipna.Subnet(sub)
	if err != nil {
		return result
	}
	for _, item := range extendedIPs {
		ip := item.String()
		if sub == 32 {
			ip = item.IP.String()
		}
		if port == "" || sub != 32 {
			result = append(result, ip)
		} else {
			for _, p := range ports {
				ipWithPort := fmt.Sprintf("%s:%s", ip, p)
				result = append(result, ipWithPort)
			}
		}
	}
	return result
}

func genPorts(port string) []string {
	switch port {
	case "small", "s":
		return pSmall

	case "medium", "m":
		return pMedium

	case "large", "l":
		return pLarge

	case "xlarge", "x":
		return []string{"80", "443", "81", "300", "591", "593", "832", "981", "1010", "1311", "2082", "2087", "2095", "2096", "2480", "3000", "3128", "3333", "4243", "4567", "4711", "4712", "4993", "5000", "5104", "5108", "5800", "6543", "7000", "7396", "7474", "8000", "8001", "8008", "8014", "8042", "8069", "8080", "8081", "8083", "8088", "8090", "8091", "8118", "8123", "8172", "8222", "8243", "8280", "8281", "8333", "8443", "8500", "8834", "8880", "8888", "8983", "9000", "9043", "9060", "9080", "9090", "9091", "9200", "9443", "9800", "9981", "12443", "16080", "18091", "18092", "20720", "28017"}
	case "full":
		var ports []string
		for i := 1; i <= 65535; i++ {
			ports = append(ports, fmt.Sprintf("%s", i))
		}
		return ports
	case "f":
		var ports []string
		for i := 1; i <= 65535; i++ {
			ports = append(ports, fmt.Sprintf("%d", i))
		}
		return ports
	default:
		return []string{"80", "443", "8000", "8080", "8443"}
	}
}
