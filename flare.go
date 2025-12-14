package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/coalaura/plain"
)

type CloudflareClient struct {
	cfg *Config
}

type CloudflareDNSRecordsResponse struct {
	Result []CloudflareDNSRecord `json:"result"`
}

type CloudflareDNSRecordResponse struct {
	Result CloudflareDNSRecord `json:"result"`
}

type CloudflareDNSRecord struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	TTL     int64  `json:"ttl"`
	Type    string `json:"type"`
	Comment string `json:"comment"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

func NewCloudflareClient(cfg *Config) *CloudflareClient {
	return &CloudflareClient{
		cfg: cfg,
	}
}

func (r *CloudflareDNSRecord) Update(ip string) {
	now := time.Now()

	r.Comment = fmt.Sprintf("Last updated: %s", now.Format(plain.RFC3339Local))
	r.Content = ip

	r.Proxied = false
}

func (c *CloudflareClient) Request(method, url string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.Key))

	return req, nil
}

func (c *CloudflareClient) FetchIP() (string, error) {
	resp, err := http.Get("https://ip.shrt.day")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := net.ParseIP(string(body))
	if ip == nil {
		return "", errors.New("invalid ip")
	}

	return ip.String(), nil
}

func (c *CloudflareClient) FindDNS(ip string) (*CloudflareDNSRecord, error) {
	query := url.Values{
		"type": []string{"A"},
		"name": []string{c.cfg.Record},
	}

	uri := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?%s", c.cfg.Zone, query.Encode())

	req, err := c.Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	var records CloudflareDNSRecordsResponse

	err = json.NewDecoder(resp.Body).Decode(&records)
	if err != nil {
		return nil, err
	}

	if len(records.Result) != 1 {
		return nil, nil
	}

	return &records.Result[0], nil
}

func (c *CloudflareClient) CreateDNS(record *CloudflareDNSRecord) (*CloudflareDNSRecord, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	req, err := c.Request("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", c.cfg.Zone), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	var created CloudflareDNSRecordResponse

	err = json.NewDecoder(resp.Body).Decode(&created)
	if err != nil {
		return nil, err
	}

	return &created.Result, nil
}

func (c *CloudflareClient) UpdateDNS(record *CloudflareDNSRecord) (*CloudflareDNSRecord, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	req, err := c.Request("PUT", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.cfg.Zone, record.ID), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	var updated CloudflareDNSRecordResponse

	err = json.NewDecoder(resp.Body).Decode(&updated)
	if err != nil {
		return nil, err
	}

	return &updated.Result, nil
}

func (c *CloudflareClient) Update(ip string) (*CloudflareDNSRecord, error) {
	log.Println("Finding record...")

	record, err := c.FindDNS(ip)
	if err != nil {
		return nil, err
	}

	if record == nil {
		log.Println("Record not found, creating...")

		record = &CloudflareDNSRecord{
			Name: c.cfg.Record,
			TTL:  60,
			Type: "A",
		}

		record.Update(ip)

		record, err = c.CreateDNS(record)
		if err != nil {
			return nil, err
		}

		log.Printf("Created record with ID %q\n", record.ID)
	} else {
		if record.Content == ip && !record.Proxied {
			log.Println("Record found, still up-to-date")

			return record, nil
		}

		log.Println("Found record, updating...")

		record.Update(ip)

		record, err = c.UpdateDNS(record)
		if err != nil {
			return nil, err
		}

		log.Printf("Updated record with ID %q\n", record.ID)
	}

	return record, nil
}

func (c *CloudflareClient) Loop() error {
	ip, err := c.FetchIP()
	if err != nil {
		return err
	}

	dns, err := c.Update(ip)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		ip, err := c.FetchIP()
		if err != nil {
			log.Warnf("Failed to fetch ip: %v\n", err)

			continue
		}

		if dns.Content == ip {
			continue
		}

		log.Printf("IP changed from %q to %q\n", dns.Content, ip)

		dns, err = c.Update(ip)
		if err != nil {
			log.Warnf("Failed to update record: %v\n", err)
		}
	}

	return nil
}
