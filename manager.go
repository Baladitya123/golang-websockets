package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		// CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients clientList
	sync.RWMutex
	handlers map[string]eventHandler // giving all eventhandlers to the manager
	otps     RetentionMap
}

func newManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(clientList),
		handlers: make(map[string]eventHandler),
		otps:     newRetentionMap(ctx, 5*time.Second),
	}
	m.setUpEventHandlers()
	return m
}
func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("there is no such event type")
	}
}

func (m *Manager) setUpEventHandlers() {
	m.handlers[EventSendMessage] = sendMessage
	m.handlers[EventChangeChatRoom] = chatRoomHandler
}
func chatRoomHandler(event Event, c *Client) error {
	var changeChatRoom ChangeChatRoom
	if err := json.Unmarshal(event.Payload, &changeChatRoom); err != nil {
		return fmt.Errorf("bad payload in the request : %v", err)
	}
	c.chatroom = changeChatRoom.Name
	return nil
}
func sendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent
	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("bad payload in the request: %v", err)
	}
	var broadcastMessage NewMessageEvent
	broadcastMessage.Sent = time.Now()
	broadcastMessage.From = chatevent.From
	broadcastMessage.Message = chatevent.Message

	data, err := json.Marshal(broadcastMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal the broadcast : %v", err)
	}
	outGoingEvent := Event{
		Payload: data,
		Type:    EventNewMessage,
	}
	for clients := range c.Manager.clients {
		if c.chatroom == clients.chatroom && clients.name != c.name {
			clients.egress <- &outGoingEvent
		}

	}
	return nil

}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req userLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	names := []string{"balu", "yaswanth", "bhagya lakshmi", "gopala krishna"}
	for _, name := range names {
		if req.Username == name && req.Password == "123" {
			type response struct {
				OTP string `json:"otp"`
			}
			otp := m.otps.newOtp()
			res := response{
				OTP: otp.key,
			}
			data, err := json.Marshal(res)
			if err != nil {
				log.Println(err)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			break
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	name := r.URL.Query().Get("name")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !m.otps.verifyOtp(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	log.Println("new connection")
	// upgrade the http connection to the websocket connection
	conn, err := websocketUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print(err)
		return
	}
	client := newClient(conn, m, name)
	m.addClient(client)
	//start client processes

	go client.readMessage()
	go client.writeMessage()

}

func (m *Manager) addClient(c *Client) {
	m.Lock()
	defer m.Unlock()
	m.clients[c] = true
}
func (m *Manager) removeClient(c *Client) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.clients[c]; ok {
		c.connection.Close()
		delete(m.clients, c)
	}

}

// and we want connection from a specific origin or region because to reduce the cross-region connections//
// and it is the job of manager

// func checkOrigin(r *http.Request) bool {
// 	origin := r.Header.Get("Origin")
// 	switch origin {
// 	case "https://localhost:8080":
// 		return true
// 	default:
// 		return false
// 	}
// }
