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
		APIKey:       os.Getenv("PORKBUN_API_KEY"),
		SecretAPIKey: os.Getenv("PORKBUN_API_SECRET"),
	})

	resp, err := client.Ping(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("HTTP Status Code: %v\n", resp.HTTPResponse.StatusCode)
	fmt.Printf("HTTP Status: %v\n", resp.HTTPResponse.Status)
	fmt.Printf("HTTP Proto: %v\n", resp.HTTPResponse.Proto)

	// Absent when the client was pointed at a plain HTTP endpoint, such as a
	// local mock through Options.BaseURL.
	if state := resp.HTTPResponse.TLS; state != nil {
		fmt.Printf("HTTP TLS Version: %v\n", tls.VersionName(state.Version))
		fmt.Printf("HTTP TLS Protocol: %v\n", state.NegotiatedProtocol)
	}

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

		dnsResp, err := client.DNS.ListRecords(context.Background(), domain.Domain)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Found %v records\n", len(dnsResp.Records))
		slices.SortFunc(dnsResp.Records, func(a, b porkbun.DNSRecord) int {
			return cmp.Compare(a.Name, b.Name)
		})

		for _, dns := range dnsResp.Records {
			// TrimSuffix, not ReplaceAll: the domain has to come off the end,
			// or a record named "example.com.example.com" loses the wrong half.
			dnsName := strings.TrimSuffix(dns.Name, "."+domain.Domain)
			if strings.EqualFold(dns.Name, domain.Domain) {
				dnsName = "@"
			}
			fmt.Printf("%v\t%v\tIN\t%v\t%v\n", dnsName, dns.TTL, dns.Type, dns.Content)
		}

		dnssecResp, err := client.DNS.GetDNSSECRecords(context.Background(), domain.Domain)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Found %v DNSSEC Records\n", len(dnssecResp.Records))

		for _, record := range dnssecResp.Records {
			fmt.Printf("KeyTag=%v\tAlg=%v\tAlgIsValid=%v\tDigestType=%v\tDigest=%v\n", record.KeyTag, record.Alg, record.Alg.IsValid(), record.DigestType, record.Digest)
		}

		fmt.Println()
	}
}
