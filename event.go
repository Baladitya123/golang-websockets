// this is used when the fronted want the data in event structure before event just the message is printed in fronted
// the frontend want the message as a event class type and also the event object will come to server as well

package main

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"` // it means payload is a type of object
}

type eventHandler func(event Event, c *Client) error

const (
	EventSendMessage    = "send_message"
	EventNewMessage     = "new_message"
	EventChangeChatRoom = "change_chatroom"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type NewMessageEvent struct {
	SendMessageEvent
	Sent time.Time `json:"sent"`
}

type ChangeChatRoom struct {
	Name string `json:"name"`
}
