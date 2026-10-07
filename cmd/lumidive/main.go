// Package main provides the entrypoint for the lumidive CLI and API server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
	"github.com/AobaIwaki123/lumidive/pkg/server"
	"github.com/AobaIwaki123/lumidive/pkg/ticketdive"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "parse":
		runParse(os.Args[2:])
	case "ical":
		runICal(os.Args[2:])
	case "server":
		runServer(os.Args[2:])
	case "version":
		fmt.Println("lumidive v1.0.0")
	case "help", "-h", "--help":
		printUsage()
	default:
		// If argument looks like a URL or ID, parse it directly
		runParse([]string{command})
	}
}

func printUsage() {
	fmt.Println("lumidive - TicketDive Adapter, CLI & API Server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lumidive parse <url-or-id>     Parse and print event metadata in JSON")
	fmt.Println("  lumidive ical <url-or-id>      Generate and print iCalendar (.ics) format")
	fmt.Println("  lumidive server [options]      Start the HTTP API server")
	fmt.Println("  lumidive version               Show version")
	fmt.Println()
	fmt.Println("Server Options:")
	fmt.Println("  --port <port>       Port to listen on (default: 8080 or $PORT)")
	fmt.Println("  --cache-ttl <dur>   Cache duration (default: 60s)")
}

func runParse(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing event URL or ID")
		fmt.Fprintln(os.Stderr, "Usage: lumidive parse <url-or-id>")
		os.Exit(1)
	}

	target := args[0]
	client := ticketdive.NewClient()
	res, err := client.FetchEvent(context.Background(), target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(res.Event); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func runICal(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing event URL or ID")
		os.Exit(1)
	}

	target := args[0]
	client := ticketdive.NewClient()
	res, err := client.FetchEvent(context.Background(), target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cal, err := ticketdive.GenerateICal(res.Event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating iCal: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(cal)
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	portDefault := os.Getenv("PORT")
	if portDefault == "" {
		portDefault = "8080"
	}

	port := fs.String("port", portDefault, "port to listen on")
	cacheTTL := fs.Duration("cache-ttl", 60*time.Second, "in-memory cache TTL")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Invalid flags: %v", err)
	}

	client := ticketdive.NewClient()
	service := ticketdive.NewCachedService(client, *cacheTTL)
	apiServer := server.NewServer(service)

	mux := http.NewServeMux()
	handler := api.HandlerFromMux(apiServer, mux)

	// Wrap with basic CORS middleware
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		handler.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:              ":" + *port,
		Handler:           corsHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("lumidive API server listening on http://localhost:%s (cache TTL: %v)", *port, *cacheTTL)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
}
