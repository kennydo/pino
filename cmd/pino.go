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
		log.Fatal("Config path must be specified!")
	}

	fmt.Printf("Loading config from: %v\n", *configPath)

	parsedConfig := pino.LoadConfig(*configPath)

	fmt.Printf("Parsed config: %#v\n", parsedConfig)
}
