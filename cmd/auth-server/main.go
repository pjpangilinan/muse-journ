package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/pjpangilinan/muse-journ/internal/spotify"
	"golang.org/x/oauth2"
)

func main() {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET required")
	}

	state := randomState()
	config := spotify.NewOAuthConfig(clientID, clientSecret, "http://127.0.0.1:9090/callback")
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	log.Println("Open this URL in your browser:")
	log.Println(authURL)

	var token *oauth2.Token
	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		var err error
		token, err = config.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, fmt.Sprintf("token exchange: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
		})

		log.Println("OAuth complete. Refresh token returned in JSON response.")
		log.Println("Save it as SPOTIFY_REFRESH_TOKEN in GitHub Secrets.")

		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt)
			<-sig
			os.Exit(0)
		}()
	})

	server := &http.Server{Addr: ":9090", Handler: mux}
	log.Printf("Server listening on http://127.0.0.1:9090")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func() {
		<-sig
		server.Shutdown(context.Background())
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
