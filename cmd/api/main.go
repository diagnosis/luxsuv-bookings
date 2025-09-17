package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("⚠️  DEPRECATED: This monolithic API is being replaced by microservices.")
	fmt.Println("🔄 Please use the new services:")
	fmt.Println("   - Gateway Service: make dev (starts all services)")
	fmt.Println("   - Auth Service: http://localhost:8081")
	fmt.Println("   - Bookings Service: http://localhost:8082")
	fmt.Println("   - All traffic should go through Gateway: http://localhost:8080")
	fmt.Println("")
	fmt.Println("🚀 To start the new microservices stack: make dev")
	os.Exit(1)
}
