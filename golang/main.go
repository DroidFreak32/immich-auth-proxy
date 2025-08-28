package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"

)

var (
	upstreamURL *url.URL
	tokenSource oauth2.TokenSource
	httpClient  *http.Client
)

func main() {
	// 1. Get configuration from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	upstreamServerURL := os.Getenv("UPSTREAM_SERVER_URL")
	if upstreamServerURL == "" {
		log.Fatal("UPSTREAM_SERVER_URL environment variable not set.")
	}

	var err error
	upstreamURL, err = url.Parse(upstreamServerURL)
	if err != nil {
		log.Fatalf("Invalid UPSTREAM_SERVER_URL: %v", err)
	}

	// 2. Initialize Google ID Token source using Application Default Credentials
	// The audience is the root URL of the service we are calling.
	ctx := context.Background()
	tokenSource, err = idtoken.NewTokenSource(ctx, upstreamServerURL)
	if err != nil {
		log.Fatalf("Failed to create ID token source: %v", err)
	}

	// 3. Create a single HTTP client to reuse for all forwarded requests
	httpClient = &http.Client{}

	// 4. Set up the proxy handler and start the server
	http.HandleFunc("/", handleProxyRequest)
	log.Printf("Starting auth proxy on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	// 1. Construct the destination URL
	// The original request's path and query are preserved.
	destinationURL := upstreamURL.ResolveReference(r.URL)
	log.Printf("Forwarding request to: %s", destinationURL.String())

	// 2. Create the forwarded request
	// The original request body is passed through.
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, destinationURL.String(), r.Body)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Copy headers and add the Authorization token
	proxyReq.Header = r.Header.Clone()
	
	// Get the Google-signed ID token
	token, err := tokenSource.Token()
	if err != nil {
		log.Printf("Error getting ID token: %v", err)
		http.Error(w, "Could not generate authentication token.", http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	
	// Set the Host header to the destination's host
	proxyReq.Host = destinationURL.Host

	// 4. Send the request to the upstream server
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 5. Copy the response back to the original client
	copyResponse(w, resp)
}

func copyResponse(w http.ResponseWriter, resp *http.Response) {
	// Copy headers from the upstream response to the client response
	for key, values := range resp.Header {
		// These headers are managed by the server; copying them can cause issues.
		if strings.ToLower(key) == "content-encoding" ||
			strings.ToLower(key) == "content-length" ||
			strings.ToLower(key) == "transfer-encoding" ||
			strings.ToLower(key) == "connection" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy the status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}
