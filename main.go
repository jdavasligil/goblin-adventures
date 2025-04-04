/*
Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
Modified 2025 J. Davasligil

Licensed under the Apache License, Version 2.0 (the "License"). You may not use
this file except in compliance with the License. A copy of the License is
located at

	http://aws.amazon.com/apache2.0/

	or in the "license" file accompanying this file. This file is distributed
	on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
	express or implied. See the License for the specific language governing
	permissions and limitations under the License.
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"

	"github.com/gorilla/websocket"
)

var (
	clientID string
	clientSecret string
	oauth2Config *clientcredentials.Config
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type JSONResponse struct {
	Data []map[string]any `json:"data"`
}

// Echo server
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Printf("Received: %s\n", message);
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println(err)
		}
	}
}

func getEnvironmentVariables() map[string]string {
	envVars := make(map[string]string)

	dat, err := os.ReadFile(".env")
	if err != nil {
		log.Fatalln("Fatal Error! Failure to read .env")
	}

	lines := bytes.SplitSeq(dat, []byte("\n"))
	for line := range lines {
		nameVar := bytes.Split(line, []byte("="))
		if len(nameVar) == 2 {
			envVars[string(nameVar[0])] = string(nameVar[1])
		}
	}

	return envVars
}

func main() {
	envVars := getEnvironmentVariables()

	clientID = envVars["CLIENT_ID"]
	if clientID == "" {
		log.Fatalln("Fatal Error! CLIENT_ID is not present in .env")
	}
	clientSecret = envVars["CLIENT_SECRET"]
	if clientSecret == "" {
		log.Fatalln("Fatal Error! CLIENT_SECRET is not present in .env")
	}

	oauth2Config = &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
		Scopes: []string{"user:read:chat", "user:write:chat"},
	}

	token, err := oauth2Config.Token(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Access Token:  %s\n", token.AccessToken)
	fmt.Printf("Token Expires: %s\n", token.Expiry.Local().Truncate(1))

	reqURL, err := url.Parse("https://api.twitch.tv/helix/users?login=twitchdev")
	if err != nil {
		log.Fatal(err)
	}
	getUserReq := &http.Request {
		Method: "GET",
		URL: reqURL,
		Header: map[string][]string{
			"Client-Id": {clientID},
		},
	}
	token.SetAuthHeader(getUserReq)
	response, err := http.DefaultClient.Do(getUserReq); if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	jsonResponse := &JSONResponse{}
	err = json.Unmarshal(body, jsonResponse)
	if err != nil {
		log.Fatal(err)
	}
	resMap := jsonResponse.Data[0]
	fmt.Println(resMap["broadcaster_type"])
	fmt.Println(resMap["description"])

	http.HandleFunc("/ws", wsHandler)
	fmt.Println("WebSocket server started on :3000")
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
