package main

import (
	"cmp"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/tuzzmaniandevil/porkbun-go"
)

func main() {
	client := porkbun.NewClient(&porkbun.Options{
		ApiKey:       os.Getenv("PORKBUN_API_KEY"),
		SecretApiKey: os.Getenv("PORKBUN_API_SECRET"),
	})

	resp, err := client.Ping(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("HTTP Status Code: %v\n", resp.HTTPResponse.StatusCode)
	fmt.Printf("HTTP Status: %v\n", resp.HTTPResponse.Status)
	fmt.Printf("HTTP Proto: %v\n", resp.HTTPResponse.Proto)
	fmt.Printf("HTTP TLS Version: %v\n", tls.VersionName(resp.HTTPResponse.TLS.Version))
	fmt.Printf("HTTP TLS Protocol: %v\n", resp.HTTPResponse.TLS.NegotiatedProtocol)
	fmt.Printf("API Status: %v\n", resp.Status)
	fmt.Printf("Your IP: %v\n", resp.YourIP)

	listDomainsResp, err := client.Domains.ListDomains(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println()
	fmt.Printf("Received %v domains\n", len(listDomainsResp.Domains))
	for _, domain := range listDomainsResp.Domains {
		fmt.Printf("Domain: %v\n", domain.Domain)

		dnsResp, err := client.Dns.GetRecords(context.Background(), domain.Domain, nil)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Found %v records\n", len(dnsResp.Records))
		slices.SortFunc(dnsResp.Records, func(a, b porkbun.DnsRecord) int {
			return cmp.Compare(a.Name, b.Name)
		})

		for _, dns := range dnsResp.Records {
			var dnsName string
			if strings.EqualFold(dns.Name, domain.Domain) {
				dnsName = "@"
			} else {
				dnsName = strings.Replace(dns.Name, "."+domain.Domain, "", -1)
			}
			fmt.Printf("%v\t%v\tIN\t%v\t%v\n", dnsName, dns.TTL, dns.Type, dns.Content)
		}

		fmt.Println()
	}
}
