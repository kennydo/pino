package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/kennydo/pino"
)

func main() {
	configPath := flag.String("config", "", "The path to a Pino config file (YAML)")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Config path must be specified!\n")
	}

	fmt.Printf("Loading config from: %v\n", *configPath)

	parsedConfig, err := pino.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Could not load config: %v\n", err)
	}

	fmt.Printf("Successfully parsed config\n")

	p, err := pino.NewPino(parsedConfig)
	if err != nil {
		log.Fatalf("Could not create Pino: %v\n", err)
	}
	p.Run()
}
