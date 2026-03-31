package main

import (
	"log"
	"os"

	"github.com/jthagar/covlet/backend/internal/server"
)

func main() {
	addr := server.AddrFromEnv()
	if err := server.Listen(addr); err != nil {
		log.Printf("server: %v", err)
		os.Exit(1)
	}
}
