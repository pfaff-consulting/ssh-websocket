package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Ssh struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"ssh"`

	Websocket struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"websocket"`
}

func loadConfig() *Config {
	paths := []string{
		"./config.yaml",
		"/etc/ssh-websocket/config.yaml",
	}

	var cfg Config
	var configPath string

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		log.Println("Config file not found in any of available paths:")
		for _, p := range paths {
			log.Println(p)
		}
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error while reading config file: %s", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Error while parsing config file: %s", err)
	}

	fmt.Printf("Config loaded from: %s\n", configPath)

	return &cfg
}

func main() {
	config := loadConfig()

	http.HandleFunc("/ssh", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, config)
	})

	fmt.Printf("Starting HTTP server on %s:%d\n", config.Websocket.Host, config.Websocket.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", config.Websocket.Host, config.Websocket.Port), nil))
}
