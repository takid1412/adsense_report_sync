package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
)

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var token oauth2.Token
	err = json.NewDecoder(f).Decode(&token)
	return &token, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func GetClient(config *oauth2.Config) *http.Client {
	tokenFile := "token.json"

	token, err := tokenFromFile(tokenFile)
	if err != nil {
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)
		var authCode string
		if _, err := fmt.Scan(&authCode); err != nil {
			log.Fatalf("Unable to read authorization code: %v", err)
		}
		token, err = config.Exchange(context.TODO(), authCode)
		if err != nil {
			fmt.Printf("Unable to retrieve token from web: %v", err)
		}
		saveToken(tokenFile, token)
	}
	return config.Client(context.Background(), token)
}
