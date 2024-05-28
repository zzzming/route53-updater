package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Route32Config struct {
	HostedZoneID string
	RecordName   string
	TTL          int64
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	} else if value == "" && defaultValue == "" {
		panic(fmt.Errorf("missing mandatory env %s", key))
	}
	return defaultValue
}

func LoadConfig() Route32Config {
	ttl, err := strconv.ParseInt(getEnv("TTL", "60"), 10, 64)
	if err != nil {
		panic(err)
	}
	return Route32Config{
		HostedZoneID: getEnv("HOSTED_ZONE_ID", ""),
		RecordName:   getEnv("RECORD_NAME", ""),
		TTL:          ttl,
	}
}

// getLocalIP retrieves the IP address of the machine where the process is running
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String(), nil
		}
	}
	return "", fmt.Errorf("unable to determine local IP")
}

// updateRoute53Record updates the Route 53 DNS record with the given name, IP, and TTL
func updateRoute53Record(hostedZoneID, recordName, ip string, ttl int64) error {
	sess := session.Must(session.NewSession())
	svc := route53.New(sess)

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: aws.String("A"),
						TTL:  aws.Int64(ttl),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ip),
							},
						},
					},
				},
			},
		},
	}

	_, err := svc.ChangeResourceRecordSets(input)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	config := LoadConfig()

	ip, err := getLocalIP()
	if err != nil {
		log.Fatalf("Failed to get local IP: %v", err)
	}
	log.Printf("Local IP: %s", ip)

	err = updateRoute53Record(config.HostedZoneID, config.RecordName, ip, config.TTL)
	if err != nil {
		log.Fatalf("Failed to update Route 53 DNS record: %v", err)
	}

	fmt.Printf("Successfully updated Route 53 DNS record for %s to %s with TTL %d\n",
		config.RecordName, ip, config.TTL)
}
