/*
Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at

	http://aws.amazon.com/apache2.0/

	or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

*/

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
)

var (
	clientID = "ui5s8b0hvlw4uz6fzd752kbvk6zipj"
	// Consider storing the secret in an environment variable or a dedicated storage system.
	clientSecret = ""
	oauth2Config *clientcredentials.Config
)

func getEnvironmentVariables() map[string]string {
	envVars := make(map[string]string)

	dat, err := os.ReadFile(".env")
	if err != nil {
		log.Fatalln("Fatal Error! Failure to read .env")
	}

	lines := bytes.Split(dat, []byte("\n"))
	for _, line := range lines {
		nameVar := bytes.Split(line, []byte("="))
		if len(nameVar) == 2 {
			envVars[string(nameVar[0])] = string(nameVar[1])
		}
	}

	return envVars
}

func main() {
	envVars := getEnvironmentVariables()

	clientSecret = envVars["CLIENT_SECRET"]
	if clientSecret == "" {
		log.Fatalln("Fatal Error! CLIENT_SECRET is not present in .env")
	}

	oauth2Config = &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
	}

	token, err := oauth2Config.Token(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Access token: %s\n", token.AccessToken)
}
