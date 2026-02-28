package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunConsole starts the interactive Gails console.
func RunConsole() {
	fmt.Println("┌───────────────────────────────────────┐")
	fmt.Println("│  🔧 Gails Console v1.0.0              │")
	fmt.Println("│  Type 'help' for available commands    │")
	fmt.Println("│  Type 'exit' or 'quit' to exit         │")
	fmt.Println("└───────────────────────────────────────┘")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("gails> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  routes  — list registered routes")
			fmt.Println("  config  — show loaded config")
			fmt.Println("  env     — show current environment")
			fmt.Println("  exit    — exit the console")
		case "env":
			env := os.Getenv("APP_ENV")
			if env == "" {
				env = "development"
			}
			fmt.Printf("Environment: %s\n", env)
		case "routes":
			fmt.Println("(Boot the app and call app.Router.Inspect())")
		case "config":
			fmt.Println("(Boot the app and inspect app.Config)")
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", line)
		}
	}
}
