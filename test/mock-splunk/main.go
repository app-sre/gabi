package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Token storage
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string // tokenName -> token
}

var (
	tokenStore = &TokenStore{
		tokens: make(map[string]string),
	}
)

// Splunk HEC event structure
type SplunkEvent struct {
	Event      interface{} `json:"event"`
	Index      string      `json:"index,omitempty"`
	Host       string      `json:"host,omitempty"`
	Source     string      `json:"source,omitempty"`
	SourceType string      `json:"sourcetype,omitempty"`
	Time       int64       `json:"time,omitempty"`
}

// Standard Splunk HEC response
type SplunkHECResponse struct {
	Text string `json:"text"`
	Code int    `json:"code"`
}

// Token creation response
type TokenCreateResponse struct {
	Entry []struct {
		Name    string `json:"name"`
		Content struct {
			Token string `json:"token"`
		} `json:"content"`
	} `json:"entry"`
}

func main() {
	// Generate certificates FIRST (synchronously, before starting servers)
	if err := generateCertificates(); err != nil {
		log.Fatalf("Failed to generate certificates: %v", err)
	}

	r := mux.NewRouter()

	// HEC endpoints
	r.HandleFunc("/services/collector/event", handleHECEvent).Methods("POST")
	r.HandleFunc("/services/collector/health/1.0", handleHealth).Methods("GET")

	// Management API endpoints (for token creation/deletion)
	r.HandleFunc("/servicesNS/admin/splunk_httpinput/data/inputs/http", handleCreateToken).Methods("POST")
	r.HandleFunc("/servicesNS/admin/splunk_httpinput/data/inputs/http/{tokenName}", handleDeleteToken).Methods("DELETE")

	// Start servers on both ports (HEC on 8088, Management API on 8089)
	go func() {
		log.Println("Starting mock Splunk HEC server on :8088")
		if err := http.ListenAndServe(":8088", r); err != nil {
			log.Fatalf("HEC server failed: %v", err)
		}
	}()

	log.Println("Starting mock Splunk Management API server on :8089")
	if err := http.ListenAndServeTLS(":8089", "/tmp/cert.pem", "/tmp/key.pem", r); err != nil {
		log.Fatalf("Management API server failed: %v", err)
	}
}

// handleHECEvent handles incoming HEC events
func handleHECEvent(w http.ResponseWriter, r *http.Request) {
	// Validate Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Println("Missing Authorization header")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SplunkHECResponse{
			Text: "Token is required",
			Code: 2,
		})
		return
	}

	// Extract token from "Splunk <token>" format
	token := strings.TrimPrefix(authHeader, "Splunk ")
	if token == authHeader {
		log.Println("Invalid Authorization header format")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SplunkHECResponse{
			Text: "Invalid authorization",
			Code: 3,
		})
		return
	}

	// Read and parse the event
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SplunkHECResponse{
			Text: "Invalid request",
			Code: 5,
		})
		return
	}

	var event SplunkEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Error parsing event JSON: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SplunkHECResponse{
			Text: "Invalid data format",
			Code: 6,
		})
		return
	}

	// Log the received event for debugging
	log.Printf("Received event with token %s: %+v", token, event)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SplunkHECResponse{
		Text: "Success",
		Code: 0,
	})
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"text":   "HEC is healthy",
		"status": "green",
	})
}

// handleCreateToken handles token creation requests
func handleCreateToken(w http.ResponseWriter, r *http.Request) {
	// Check Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok || username != "admin" {
		log.Println("Invalid or missing Basic Auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	log.Printf("Token creation requested by %s with password %s", username, password)

	// Parse form data to get token name
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tokenName := r.FormValue("name")
	if tokenName == "" {
		log.Println("Missing token name")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate a new token UUID
	newToken := uuid.New().String()

	// Store the token
	tokenStore.mu.Lock()
	tokenStore.tokens[tokenName] = newToken
	tokenStore.mu.Unlock()

	log.Printf("Created token '%s': %s", tokenName, newToken)

	// Return token in Splunk format
	response := TokenCreateResponse{
		Entry: []struct {
			Name    string `json:"name"`
			Content struct {
				Token string `json:"token"`
			} `json:"content"`
		}{
			{
				Name: tokenName,
				Content: struct {
					Token string `json:"token"`
				}{
					Token: newToken,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleDeleteToken handles token deletion requests
func handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	// Check Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok || username != "admin" {
		log.Println("Invalid or missing Basic Auth")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	log.Printf("Token deletion requested by %s with password %s", username, password)

	vars := mux.Vars(r)
	tokenName := vars["tokenName"]

	tokenStore.mu.Lock()
	_, exists := tokenStore.tokens[tokenName]
	if exists {
		delete(tokenStore.tokens, tokenName)
		log.Printf("Deleted token: %s", tokenName)
	}
	tokenStore.mu.Unlock()

	if !exists {
		log.Printf("Token not found: %s", tokenName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// generateCertificates creates self-signed certificates for HTTPS
func generateCertificates() error {
	// Minimal self-signed certificate (valid for localhost)
	cert := `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUFU4D0dNBX/8oftu1uEPjWCkMdH8wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI1MTAzMTE2MDQyNVoXDTM1MTAy
OTE2MDQyNVowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAq4oKL0EzG+XlN0VKnkHHRIaQomvTt4qXlegi3R1B+PWe
XOuu3ow1OT98bAnKBn8R/p2En3NJpwNJXMZ/FBwBhYacL7v7O7e0cGo1ZjFjYUV/
3ghHjMDddsoEg3xRnhiXPVwF206Wt3UWs4asuTSyGXcE8yywkaGfMuBKlb84kYuz
F7oIvQW0WkqFv4sHiK+jq4+98F02prp02Mv0Y6afQdJ1NoK0WLiUWobXash95Y3b
O5y/m/qnTn97CVvtBtpmIOCPRwyYU1DnHAnSMLsIvEM8seZFl0JGvYKlHcMIynnS
zu9ruETno3MUcifd9jNtwJLx8apdobNp6Zgnypq1tQIDAQABo1MwUTAdBgNVHQ4E
FgQUQPl7PpsmElGJx4Bss6aWsSYb14IwHwYDVR0jBBgwFoAUQPl7PpsmElGJx4Bs
s6aWsSYb14IwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAdgbr
m+BXhKnEIJLbL2h6z5fbeL1rP+iWnlnZHHmR5Vty0S2rnyp1NCigLTRrYiev5g4/
ybuoYm3qL2U7FE7D64BtEy+DKOCNv/XlgNEuIulXuuHdfMg7k975vLpNMm8RtZuF
wVNqxSYNhKYNOPXcZx0mBTm7D0XmF5bOuDViiEV747YZg9c9GzwNqHx50EpFBnVp
gjlq0Lo5IitOaw4IRzrF7mg4Z3rPOoTeFWlKl05H83q3x85Y3oE24ZjmoQ8Ymqzc
vKaCn3hGfJSCNWwpnLFftYnRHNCtGvSMD5fRK7wA5Sp9js0LegvO7bsP0yQlmweG
FrUYOfpwTq7UMSCTPA==
-----END CERTIFICATE-----`

	key := `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCrigovQTMb5eU3
RUqeQcdEhpCia9O3ipeV6CLdHUH49Z5c667ejDU5P3xsCcoGfxH+nYSfc0mnA0lc
xn8UHAGFhpwvu/s7t7RwajVmMWNhRX/eCEeMwN12ygSDfFGeGJc9XAXbTpa3dRaz
hqy5NLIZdwTzLLCRoZ8y4EqVvziRi7MXugi9BbRaSoW/iweIr6Orj73wXTamunTY
y/Rjpp9B0nU2grRYuJRahtdqyH3ljds7nL+b+qdOf3sJW+0G2mYg4I9HDJhTUOcc
CdIwuwi8Qzyx5kWXQka9gqUdwwjKedLO72u4ROejcxRyJ932M23AkvHxql2hs2np
mCfKmrW1AgMBAAECggEACMY5kaO34covYJWghLpQH0rzzH8P+BA0g0Q5vk4wEPQ9
WrrqebM5yLkc2+jfRthKmxfDECXlQ0/5eW+k0epB0KrrQ8zNg8c4iVSBcT3++5uC
uCB7ynEWEuyv8OrTwO64k7ioiwh40J8CX4H4xUty/bb3D5o+WOCnxEIxRnoewmEJ
ttzjtvQm0VtitBRyc2KGGp27XwwVZ5VCL3P5wpRbWjG9OT8qXOsdH8B6LhyFbmeQ
MwEbEZIjs3zv+mwNEamB3t28ZSw6eCXXD5bY4aH41ohhu8yuzQz0uD53F+Jnwg/+
hi6JRes1GHPWhMo8uoHAbZgai07qgxwoYZVDMnCOwQKBgQDfyExLi3qynbQWW6AC
xo3dTftumRuFRsoA1Ei2NiqfM3axN6ldk2duOYqiP7VQ4zPLR3oJzPn6MopMSAGv
/bdb7EiMrAkUQuh4q3OF0gdCQ6Zflyj4+DE/ln+gyReHdtzY68JQvG86XTSkkitW
zOuRBcYS6nV9UTkDGrnRtSEM9QKBgQDEPEWVV5VywjRC3wdURND180oe5zEpxeoY
IebKImb8SodDPhHmLk3EN8+iRTItanCe6CRO7eVpYaG3bJBwuDh6MAq2Q9TwwyiX
kZC+sTpCVmI7YmCydR8il0aTW5muicHHOUtGDtUVe8gvfPUjWbyhuqoJlxWUYoae
9RQ6zjmNwQKBgQDECRAlAavbuuEe0OcsZ0mxe3XuHfwI1clxzoVf8bWGvyuT9YHN
sqph2dCDb7HjiRU/V38mBMVbk1ipmE66IFW3Fhc7/Bz7/dzukKtjqGipeN/PT3ka
GTnzTWDCtkaBafPvpyATX+9EPpA2NsH2iQ83fMpWmcOQo02BVbFAbx7j3QKBgQCq
fNrPdvpma4dQGciaU/df13EsDBxQeJx7Pujt8Jmc03tU1lZirxPtR7fi+U8w2eri
hDkXQeQwfBwt6epLpCGdNqP32lLkoglgNZ2VrxA4liheA4iTQuI8AUXVvJ12YES0
u3hEc5p5QcOYTm4DunEB6dTU5ChhdIAVFkPvG0AxgQKBgCDPAcd6W4H7JDjLxYNH
z3gHh89QRlSVWmHogmYjLnOliRjWXnAJyrlaDNiihtWKurDKX4Du4mXMGh3TLVjo
l/PJ2veRMCLNk4Ebt/S700vcvjyENYKZ3Z6lk2pbf4SQpfy0gAT77nXzfk2doiLq
qKr8QU3PnJicYzOtjKwwao6d
-----END PRIVATE KEY-----`

	if err := os.WriteFile("/tmp/cert.pem", []byte(cert), 0644); err != nil {
		return fmt.Errorf("failed to write cert: %w", err)
	}
	if err := os.WriteFile("/tmp/key.pem", []byte(key), 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	log.Println("Self-signed certificates generated")
	return nil
}
