package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	version = "1.0.0"
	addr    = flag.String("addr", "localhost:18888", "Node address")
	action  = flag.String("action", "", "Action: status|plugins|exec|deploy")
	plugin  = flag.String("plugin", "", "Plugin name")
	command = flag.String("cmd", "", "Shell command to execute")
	timeout = flag.Duration("timeout", 10*time.Second, "Command timeout")
)

func main() {
	flag.Parse()

	if *action == "" {
		printUsage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var err error
	switch *action {
	case "status":
		err = doStatus(ctx)
	case "plugins":
		err = doPlugins(ctx)
	case "exec":
		err = doExec(ctx, *command)
	case "connect":
		err = doConnect(ctx)
	default:
		fmt.Printf("Unknown action: %s\n", *action)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("TCP Runtime CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  anthill-cli [options] -action <action>")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Actions:")
	fmt.Println("  status    - Get node status")
	fmt.Println("  plugins   - List loaded plugins")
	fmt.Println("  exec      - Execute shell command")
	fmt.Println("  connect   - Connect to node")
}

func doStatus(ctx context.Context) error {
	url := fmt.Sprintf("ws://%s/ws", *addr)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	handshake := map[string]string{
		"type":    "handshake",
		"node_id": uuid.New().String(),
		"version": "1.0.0",
	}

	if err := conn.WriteJSON(handshake); err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := conn.ReadJSON(&resp); err != nil {
		return err
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))

	return nil
}

func doPlugins(ctx context.Context) error {
	url := fmt.Sprintf("http://%s/api/plugins", *addr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var plugins []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return err
	}

	fmt.Println("Loaded Plugins:")
	for _, p := range plugins {
		fmt.Printf("  - %s (v%s) [%s]\n", p["name"], p["version"], p["type"])
	}

	return nil
}

func doExec(ctx context.Context, cmd string) error {
	if cmd == "" {
		return fmt.Errorf("command required for exec action")
	}

	url := fmt.Sprintf("http://%s/api/exec", *addr)

	reqBody, _ := json.Marshal(map[string]string{"command": cmd})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Output string `json:"output"`
		Error  string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
	}
	fmt.Print(result.Output)

	return nil
}

func doConnect(ctx context.Context) error {
	url := fmt.Sprintf("ws://%s/connect", *addr)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("Connected. Type 'exit' to quit.")

	for {
		fmt.Print("> ")
		var input string
		fmt.Scanln(&input)

		if input == "exit" || input == "quit" {
			break
		}

		if err := conn.WriteJSON(map[string]string{"command": input}); err != nil {
			return err
		}

		var resp map[string]interface{}
		if err := conn.ReadJSON(&resp); err != nil {
			return err
		}

		fmt.Println(resp["output"])
	}

	return nil
}
