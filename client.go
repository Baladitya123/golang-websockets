package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// ping-pong connection,it is used to know that the connection is alive or not
// the frontedn will send the ping to the server and the server should send pong like , hey! i am alive if not then the connection is lost
// this is also known as heartbeat

var (
	pongWait     = time.Second * 10    // it means the server should wait maximum for 10 seconds to get the response for its ping sent to the client
	pingInterval = (9 * pongWait) / 10 //the pinginterval means for how each time the server should send ping to the client and it should be less than pongwait because if
	// if it is greater than pongwait then , the connection will be lost before getting the response
)

type clientList map[*Client]bool

type Client struct {
	name       string
	connection *websocket.Conn
	Manager    *Manager
	// egress is used to avoid concurrent writes on the websocket connection
	egress   chan *Event
	chatroom string
}

func newClient(connection *websocket.Conn, manager *Manager, name string) *Client {
	return &Client{name, connection, manager, make(chan *Event), "general"}
}

func (c *Client) readMessage() {
	// when the break statement hits
	// clear the connection
	defer func() {
		c.Manager.removeClient(c)
	}()
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil { // setting time for the pong message to receive from cvlient
		log.Println(err)
		return
	}
	c.connection.SetReadLimit(512) // we are setting the size of the message limit // 512--byte size
	c.connection.SetPongHandler(c.pongHandler)
	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading messages: ", err)
			}
			break
		}
		var request Event
		if err := json.Unmarshal(payload, &request); err != nil { // converting the json string into the object request(Event)
			log.Printf("error while unmarshaling event :%v", err)
			break
		}
		if err := c.Manager.routeEvent(request, c); err != nil {
			log.Println("error while handeling message : ", err)
		}
	}
}

func (c *Client) writeMessage() {
	// when the break statement hits
	// clear the connection
	defer func() {
		c.Manager.removeClient(c)
	}()
	ticker := time.NewTicker(pingInterval) // for each pinginterval time send a ticket to the channel of ticker
	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				// just sending the data to the clients and checks for error
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed : ", err)
				}
				return // calling defer function
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("message sent")
		case <-ticker.C:
			log.Println("ping")
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("error while writing ping message: ", err)
				return
			}

		}
	}
}
func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait)) // again we should continue to wait for next times
}
