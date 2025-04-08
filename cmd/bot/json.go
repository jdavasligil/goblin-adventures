package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type JSONResponse struct {
	Data []map[string]any `json:"data"`
}

type JSONErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type WSResponse struct {
	Metadata map[string]string `json:"metadata"`
	Payload  map[string]string `json:"payload"`
}

type WSChatMessageMeta struct {
	MessageID           string `json:"message_id"`
	MessageTimestamp    string `json:"message_timestamp"`
	MessageType         string `json:"message_type"`
	SubscriptionType    string `json:"subscription_type"`
	SubscriptionVersion string `json:"subscription_version"`
}

type WSChatMessagePayload struct {
	Event struct {
		Badges []struct {
			ID    string `json:"id"`
			Info  string `json:"info"`
			SetID string `json:"set_id"`
		} `json:"badges"`
		BroadcasterUserId           string  `json:"broadcaster_user_id"`
		BroadcasterUserLogin        string  `json:"broadcaster_user_login"`
		BroadcasterUserName         string  `json:"broadcaster_user_name"`
		ChannelPointsAnimationId    *string `json:"channel_points_animation_id"`
		ChannelPointsCustomRewardId *string `json:"channel_points_custom_reward_id"`
		ChatterUserId               string  `json:"chatter_user_id"`
		ChatterUserLogin            string  `json:"chatter_user_login"`
		ChatterUserName             string  `json:"chatter_user_name"`
		Cheer                       *string `json:"cheer"`
		Color                       string  `json:"color"`
		IsSourceOnly                *string `json:"is_source_only"`
		Message                     struct {
			Fragments []struct {
				Cheermote *string `json:"cheermote"`
				Emote     *string `json:"emote"`
				Mention   *string `json:"mention"`
				Text      string  `json:"text"`
				Type      string  `json:"type"`
			} `json:"fragments"`
			Text string `json:"text"`
		} `json:"message"`
		MessageId                  string  `json:"message_id"`
		MessageType                string  `json:"message_type"`
		Reply                      *string `json:"reply"`
		SourceBadges               *string `json:"source_badges"`
		SourceBroadcasterUserId    *string `json:"source_broadcaster_user_id"`
		SourceBroadcasterUserLogin *string `json:"source_broadcaster_user_login"`
		SourceBroadcasterUserName  *string `json:"source_broadcaster_user_name"`
		SourceMessageId            *string `json:"source_message_id"`
	} `json:"event"`

	Subscription struct {
		Condition struct {
			BroadcasterUserId string `json:"broadcaster_user_id"`
			UserId            string `json:"user_id"`
		} `json:"condition"`
		Cost      int    `json:"cost"`
		CreatedAt string `json:"created_at"`
		Id        string `json:"id"`
		Status    string `json:"status"`
		Transport struct {
			Method    string `json:"method"`
			SessionId string `json:"session_id"`
		} `json:"transport"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"subscription"`
}

type WSChatMessage struct {
	Metadata WSChatMessageMeta    `json:"metadata"`
	Payload  WSChatMessagePayload `json:"payload"`
}

type WSMessageType struct {
	Metadata struct {
		MessageType      string `json:"message_type"`
		SubscriptionType string `json:"subscription_type"`
	} `json:"metadata"`
}

type WSSessionID struct {
	Payload struct {
		Session struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"session"`
	} `json:"payload"`
}

func PrettyPrint(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(b))
}
