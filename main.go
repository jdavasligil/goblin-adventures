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
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
	// "github.com/gorilla/websocket"
)

const (
	Host   = "127.0.0.1"
	Port   = ":8000"
	Addr   = Host + Port
	BotURI = "https://" + Host + Port

	CertFile = "certs/localhost.crt"
	KeyFile  = "certs/localhost.key"

	ChatUserId    = "1293798596" //"1048391821"
	BroadcasterID = "1048391821"
	EventSubAddr  = "eventsub.wss.twitch.tv"

	ExitSuccess   int = 0
	ExitError     int = 1
	ExitTerminate int = 4
)

var (
	BotScopes = []string{
		"user:read:chat",
		"user:write:chat",
		"user:bot",
	}
)

type UserToken struct {
	oauth2.Token
	Scope []string `json:"scope"`
}

type JSONResponse struct {
	Data []map[string]any `json:"data"`
}

type JSONErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type WSResponse struct {
	Metadata map[string]any `json:"metadata"`
	Payload  map[string]any `json:"payload"`
}

// handleSignals is responsible for handling the Linux OS termination signals.
// This is necessary to gracefully exit from the server. This is done by
// passing the cancel callback function which signals services to close.
func handleSignals(cancel context.CancelFunc, signalChan chan os.Signal) {
	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
		syscall.SIGTERM, // kill -SIGTERM XXXX
	)

	// Block until signal is received.
	<-signalChan
	//log.Print("signal: os.Interrupt - shutting down...\n")

	// Notify the server to shutdown.
	cancel()
}

// Starts an http server on the given address [HOST][PORT]. Can be shut down
// with a context with Cancel safely.
func StartHTTPServer(addr string, ctx context.Context) error {
	shutdownChan := make(chan bool, 1)

	server := &http.Server{
		Addr:              addr,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
		Handler:           http.DefaultServeMux,
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := server.ListenAndServeTLS(CertFile, KeyFile); !errors.Is(err, http.ErrServerClosed) {
			log.Println("http:", err)
		}

		shutdownChan <- true
	}()

	<-ctx.Done()

	ctxShutdown, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	err := server.Shutdown(ctxShutdown)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	<-shutdownChan
	//log.Println("http:", "Server gracefully shut down.")

	return err
}

type ChatBot struct {
	UserID   string
	ClientID string

	clientSecret string
	oauth2Config *clientcredentials.Config
	token        *oauth2.Token
	userToken    *UserToken
	sessionID    string
	authCode     string

	conn *websocket.Conn
}

func (b *ChatBot) GetEnvironmentVariables() {
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
	b.ClientID = envVars["CLIENT_ID"]
	b.clientSecret = envVars["CLIENT_SECRET"]

	if b.ClientID == "" {
		log.Fatalln("Fatal Error! CLIENT_ID is not present in .env")
	}
	if b.clientSecret == "" {
		log.Fatalln("Fatal Error! CLIENT_SECRET is not present in .env")
	}
}

// Creates a link to request authorization from user. Spins up a temporary
// http server to accept a single redirect and reads the query params.
//
// #MUST be called BEFORE GetClientAuthToken
//
// Follows the Authorization code grant flow. Link:
// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#authorization-code-grant-flow
func (b *ChatBot) MakeAuthRequest() {
	var query strings.Builder

	scopeStr := strings.Join(BotScopes, " ")
	state := strings.ToLower(rand.Text())
	redirectPath := "/auth"

	query.WriteString("response_type=code")
	query.WriteString("&client_id=" + b.ClientID)
	query.WriteString("&redirect_uri=" + BotURI + redirectPath)
	query.WriteString("&scope=" + url.QueryEscape(scopeStr))
	query.WriteString("&state=" + state)

	reqURL := url.URL{
		Scheme:   "https",
		Host:     "id.twitch.tv",
		Path:     "oauth2/authorize",
		RawQuery: query.String(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	signalChan := make(chan os.Signal, 1)
	authChan := make(chan string, 3)

	defer close(signalChan)
	defer close(authChan)

	http.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		vals := r.URL.Query()

		err := vals.Get("error")
		if err != "" {
			err += ": "
			err += vals.Get("error_description")
		}

		rstate := vals.Get("state")
		if rstate != state {
			if err != "" {
				err += "\n"
			}
			err += "state_mismatch: The response state does not match"
		}

		authChan <- err
		authChan <- vals.Get("code")

		signalChan <- syscall.SIGINT
	})

	fmt.Println("Click the link below while logged in to authenticate:")
	fmt.Println(reqURL.String())

	go handleSignals(cancel, signalChan)

	err := StartHTTPServer(Port, ctx)
	if err != nil {
		log.Println("http:", err)
		return
	}

	select {
	case errStr := <-authChan:
		if errStr != "" {
			log.Print(errStr)
		}
	default:
		log.Fatal("Process interrupted.")
	}

	select {
	case code := <-authChan:
		b.authCode = code
	default:
		log.Fatal("Process interrupted.")
	}
}

func (b *ChatBot) GetUserAuthToken() {
	reqURL := &url.URL{
		Scheme: "https",
		Host:   "id.twitch.tv",
		Path:   "/oauth2/token",
	}
	fmt.Println("reqURL", reqURL.String())

	var body strings.Builder

	body.WriteString("client_id=" + b.ClientID)
	body.WriteString("&client_secret=" + b.clientSecret)
	body.WriteString("&code=" + b.authCode)
	body.WriteString("&grant_type=authorization_code")
	body.WriteString("&redirect_uri=" + BotURI)

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Body:   io.NopCloser(strings.NewReader(body.String())),
		Header: http.Header{},
	}
	defer req.Body.Close()

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Client-ID", b.ClientID)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	reqBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}

	if response.StatusCode != 200 {
		fmt.Println(string(reqBody))
		return
	}

	b.userToken = &UserToken{}

	err = json.Unmarshal(reqBody, b.userToken)
	if err != nil {
		log.Println("json:", err)
	}

	fmt.Println(b.userToken.AccessToken)
}

// Obtain a limited (server-to-server) token through the client credentials
// grant flow.
//
// https://datatracker.ietf.org/doc/html/rfc6749#section-1.3.4
// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#authorization-code-grant-flow
func (b *ChatBot) GetClientAuthToken() {
	b.oauth2Config = &clientcredentials.Config{
		ClientID:     b.ClientID,
		ClientSecret: b.clientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
		Scopes:       BotScopes,
	}

	var err error
	b.token, err = b.oauth2Config.Token(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
}

// MUST be called AFTER GetClientAuthToken
func (b *ChatBot) ValidateAuth() {
	reqURL, err := url.Parse("https://id.twitch.tv/oauth2/validate")
	if err != nil {
		log.Println(err)
		return
	}

	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: http.Header{},
	}

	b.token.SetAuthHeader(req)
	req.Header.Set("Client-ID", b.ClientID)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	resBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(string(resBody))
	// TODO Destructure json https://dev.twitch.tv/docs/authentication/validate-tokens/
	// perform proper validation
}

func (b *ChatBot) RequestUserInfo(user string) {
	reqURL, err := url.Parse("https://api.twitch.tv/helix/users?login=" + user)
	if err != nil {
		log.Println(err)
		return
	}

	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: http.Header{},
	}

	b.token.SetAuthHeader(req)
	req.Header.Set("Client-ID", b.ClientID)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}

	jsonResponse := &JSONResponse{}
	err = json.Unmarshal(body, jsonResponse)
	if err != nil {
		log.Println(err)
		return
	}
	resMap := jsonResponse.Data[0]

	PrettyPrint(resMap)
}

func (b *ChatBot) SendMessage(m string) {
	reqURL := &url.URL{
		Scheme: "https",
		Host:   "api.twitch.tv",
		Path:   "/helix/chat/messages",
	}
	fmt.Println(reqURL.String())

	body, err := json.Marshal(struct {
		BroadcasterID string `json:"broadcaster_id"`
		SenderID      string `json:"sender_id"`
		Message       string `json:"message"`
	}{
		BroadcasterID: BroadcasterID,
		SenderID:      b.UserID,
		Message:       m,
	})
	if err != nil {
		log.Println(err)
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Body:   io.NopCloser(strings.NewReader(string(body))),
		Header: http.Header{},
	}
	defer req.Body.Close()

	b.userToken.SetAuthHeader(req)
	req.Header.Set("Client-ID", b.ClientID)
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()
	reqBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}
	if response.StatusCode != 202 {
		jsonResponse := &JSONErrorResponse{}
		err = json.Unmarshal(reqBody, jsonResponse)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(jsonResponse.Status, jsonResponse.Error, jsonResponse.Message)
		return
	}
	jsonResponse := &JSONResponse{}
	err = json.Unmarshal(reqBody, jsonResponse)
	if err != nil {
		log.Println(err)
		return
	}
	resMap := jsonResponse.Data[0]
	PrettyPrint(resMap)
}

func (b *ChatBot) RegisterEventSubListeners() {
	reqURL, err := url.Parse("https://api.twitch.tv/helix/eventsub/subscriptions")
	if err != nil {
		log.Println(err)
		return
	}
	body, err := json.Marshal(struct {
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			BroadcastUserID string `json:"broadcaster_user_id"`
			UserID          string `json:"user_id"`
		} `json:"condition"`
		Transport struct {
			Method    string `json:"method"`
			SessionID string `json:"session_id"`
		} `json:"transport"`
	}{
		Type:    "channel.chat.message",
		Version: "1",
		Condition: struct {
			BroadcastUserID string `json:"broadcaster_user_id"`
			UserID          string `json:"user_id"`
		}{
			BroadcastUserID: BroadcasterID,
			UserID:          b.UserID,
		},
		Transport: struct {
			Method    string `json:"method"`
			SessionID string `json:"session_id"`
		}{
			Method:    "websocket",
			SessionID: b.sessionID,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	req := &http.Request{
		Method: http.MethodPost,
		URL:    reqURL,
		Body:   io.NopCloser(strings.NewReader(string(body))),
		Header: http.Header{},
	}
	defer req.Body.Close()
	fmt.Println(string(body))

	req.Header.Set("Client-Id", b.ClientID)
	req.Header.Set("Content-Type", "application/json")
	b.userToken.SetAuthHeader(req)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	reqBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}
	if response.StatusCode != 202 {
		jsonResponse := &JSONErrorResponse{}
		err = json.Unmarshal(reqBody, jsonResponse)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(jsonResponse.Status, jsonResponse.Error, jsonResponse.Message)
		return
	}
	//TEMP
	fmt.Println(string(reqBody))
	jsonResponse := &JSONResponse{}
	err = json.Unmarshal(reqBody, jsonResponse)
	if err != nil {
		log.Println(err)
		return
	}
	resMap := jsonResponse.Data[0]
	PrettyPrint(resMap)
}

func (b *ChatBot) HandleMessage(m []byte) {
	msg := &WSResponse{}
	err := json.Unmarshal(m, msg)
	if err != nil {
		log.Println("handle:", err)
	}
	PrettyPrint(msg)
	msgType := msg.Metadata["message_type"]
	switch msgType {
	case "session_welcome":
		session := msg.Payload["session"].(map[string]any)
		b.sessionID = session["id"].(string)
		go b.RegisterEventSubListeners()
	case "notification":
		subType := msg.Metadata["subscription_type"].(string)
		switch subType {
		case "channel.chat.message":
			event := msg.Payload["event"].(map[string]any)
			fmt.Println(event["chatter_user_login"])
		}
	}
}

func (b *ChatBot) Run() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	wsURL := url.URL{Scheme: "wss", Host: EventSubAddr, Path: "/ws"}
	fmt.Printf("Connecting to %s\n", wsURL.String())

	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	b.conn = c
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				return
			}
			b.HandleMessage(message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			fmt.Println(t)
			//err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			//if err != nil {
			//	log.Println("write:", err)
			//	return
			//}
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func PrettyPrint(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(b))
}

func main() {
	bot := &ChatBot{
		UserID: ChatUserId,
	}
	bot.GetEnvironmentVariables()
	bot.MakeAuthRequest()
	bot.GetUserAuthToken()
	//bot.GetClientAuthToken()
	//bot.ValidateAuth()
	//bot.RequestUserInfo("crashtestgoblin")
	//bot.Run()
	bot.SendMessage("Hello!")
}
