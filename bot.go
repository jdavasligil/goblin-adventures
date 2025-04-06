// TODO: Automatically refresh tokens https://dev.twitch.tv/docs/authentication/refresh-tokens/
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

type ChatBot struct {
	UserID        string
	ClientID      string
	BroadcasterID string
	sessionID     string

	clientSecret string
	oauth2Config *clientcredentials.Config

	clientToken *oauth2.Token // Only has server to server API access
	userToken   *UserToken    // Requires bot account user authentication
	authCode    string

	shutdown chan bool

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

	b.UserID = envVars["BOT_USER_ID"]
	b.ClientID = envVars["CLIENT_ID"]
	b.BroadcasterID = envVars["BROADCASTER_ID"]
	b.clientSecret = envVars["CLIENT_SECRET"]

	if b.UserID == "" {
		log.Fatalln("Fatal Error! BOT_USER_ID is not present in .env")
	} else if b.ClientID == "" {
		log.Fatalln("Fatal Error! CLIENT_ID is not present in .env")
	} else if b.BroadcasterID == "" {
		log.Fatalln("Fatal Error! BROADCASTER_ID is not present in .env")
	} else if b.clientSecret == "" {
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

	fmt.Print("Click the link below while logged on the bot account to authenticate:\n\n")
	fmt.Println(reqURL.String())

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

// Request the short lived user auth token used to control the bot.
// MUST be called AFTER MakeAuthRequest().
func (b *ChatBot) GetUserAuthToken() {
	var body strings.Builder

	body.WriteString("client_id=" + b.ClientID)
	body.WriteString("&client_secret=" + b.clientSecret)
	body.WriteString("&code=" + b.authCode)
	body.WriteString("&grant_type=authorization_code")
	body.WriteString("&redirect_uri=" + BotURI)

	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Scheme: "https",
			Host:   "id.twitch.tv",
			Path:   "/oauth2/token",
		},
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

	fmt.Print("\nbot: Token received. Expires in ", float32(b.userToken.ExpiresIn)/3600.0, " hours.\n\n")

	b.ValidateUserAuthToken()
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
	b.clientToken, err = b.oauth2Config.Token(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
}

// MUST be called AFTER GetUserAuthToken()
func (b *ChatBot) ValidateUserAuthToken() {
	if b.userToken == nil {
		log.Fatal("Error: ValidateUserAuthToken called before GetUserAuthToken")
	}

	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "id.twitch.tv",
			Path:   "/oauth2/validate",
		},
		Header: http.Header{},
	}

	b.userToken.SetAuthHeader(req)
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
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.twitch.tv",
			Path:   "/helix/users?login=" + user,
		},
		Header: http.Header{},
	}

	b.clientToken.SetAuthHeader(req)
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
	body, err := json.Marshal(struct {
		BroadcasterID string `json:"broadcaster_id"`
		SenderID      string `json:"sender_id"`
		Message       string `json:"message"`
	}{
		BroadcasterID: b.BroadcasterID,
		SenderID:      b.UserID,
		Message:       m,
	})
	if err != nil {
		log.Println(err)
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.twitch.tv",
			Path:   "/helix/chat/messages",
		},
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
			BroadcastUserID: b.BroadcasterID,
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
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.twitch.tv",
			Path:   "/helix/eventsub/subscriptions",
		},
		Body:   io.NopCloser(strings.NewReader(string(body))),
		Header: http.Header{},
	}
	defer req.Body.Close()

	// fmt.Println(string(body))

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
	//fmt.Println("MESSAGE RECV:")
	//PrettyPrint(msg)
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
			chatterUserID := event["chatter_user_id"]
			chatMessage := event["message"].(map[string]any)
			chatText := chatMessage["text"].(string)
			if chatterUserID == b.BroadcasterID {
				if strings.ToLower(chatText) == "!shutdown" {
					b.shutdown <- true
				}
			}
		}
	}
}

func (b *ChatBot) Run() {

	interrupt := make(chan os.Signal, 1)
	b.shutdown = make(chan bool, 1)

	signal.Notify(
		interrupt,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
		syscall.SIGTERM, // kill -SIGTERM XXXX
	)
	defer signal.Reset()

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
				log.Println(err)
				return
			}
			b.HandleMessage(message)

			select {
			case <-b.shutdown:
				return
			default:
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	alive := true

	for alive {
		select {
		case <-done:
			log.Println("websocket: All Done.")
			alive = false
		case <-ticker.C:
			//err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			//if err != nil {
			//	log.Println("write:", err)
			//	return
			//}
		case <-interrupt:
			log.Println("websocket: Interrupt received.")
			alive = false
			b.shutdown <- true
		}
	}

	log.Println("websocket: Shutting down.")
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println(err)
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}
