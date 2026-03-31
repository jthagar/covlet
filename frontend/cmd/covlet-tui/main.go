package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jthagar/covlet/backend/pkg/config"
	"github.com/jthagar/covlet/frontend/pkg/tui"
)

func main() {
	api := flag.String("api", getenvDefault("COVLET_API", "http://127.0.0.1:8080"), "covlet API base URL")
	resume := flag.String("resume", getenvDefault("COVLET_RESUME", ""), "path to resume YAML for rendering")
	flag.Parse()

	var r config.Resume
	if p := strings.TrimSpace(*resume); p != "" {
		cfg, err := config.LoadConfig(p)
		if err != nil {
			log.Fatalf("load resume: %v", err)
		}
		r = cfg.Resume
	}

	if err := tui.Run(*api, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

