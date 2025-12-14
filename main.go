package main

import "github.com/coalaura/plain"

var log = plain.New(plain.WithDate(plain.RFC3339Local))

func main() {
	log.Println("Loading config...")

	cfg, err := LoadConfig()
	log.MustFail(err)

	client := NewCloudflareClient(cfg)

	client.Loop()
}
