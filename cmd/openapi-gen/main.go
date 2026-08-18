package main

import (
	"encoding/json"
	"log"
	"os"

	internalapi "github.com/shubhindia/gpu-telemetry/internal/api"
)

func main() {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(internalapi.OpenAPISpec()); err != nil {
		log.Fatal(err)
	}
}
