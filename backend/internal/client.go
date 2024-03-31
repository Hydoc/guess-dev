package internal

import (
	"github.com/gorilla/websocket"
	"log"
)

const (
	ProductOwner = "product-owner"
	Developer    = "developer"
)

type userDTO map[string]interface{}

type Client struct {
	connection *websocket.Conn
	room       *Room
	Name       string
	Role       string
	Guess      int
	send       chan message
}

func newClient(name, role string, room *Room, connection *websocket.Conn) *Client {
	return &Client{
		room:       room,
		Name:       name,
		connection: connection,
		Role:       role,
		send:       make(chan message),
	}
}

func (client *Client) websocketReader() {
	defer func() {
		client.room.leave <- client
		client.room.broadcast <- newLeave()
		client.connection.Close()
	}()
	for {
		var incMessage clientMessage
		err := client.connection.ReadJSON(&incMessage)
		if err != nil {
			log.Println("error reading incoming client message:", err)
			break
		}

		switch {
		case incMessage.Type == guess && client.Role == Developer:
			actualGuess := int(incMessage.Data.(float64))
			client.Guess = actualGuess
			client.room.broadcast <- newDeveloperGuessed()
			client.send <- newYouGuessed(actualGuess)
		case incMessage.Type == newRound && client.Role == ProductOwner:
			client.room.broadcast <- newResetRound()
		case incMessage.Type == lockRoom:
			pw, pwOk := incMessage.Data.(map[string]any)["password"]
			key, keyOk := incMessage.Data.(map[string]any)["key"]

			if !keyOk {
				log.Println("client:", client.Name, "tried to lock room", client.room.id, "without a key")
				break
			}
			if !pwOk {
				log.Println("client:", client.Name, "tried to lock room", client.room.id, "without a password")
				break
			}

			if client.room.lock(client.Name, pw.(string), key.(string)) {
				client.room.broadcast <- newRoomLocked()
				break
			}
			log.Println("was not able to lock room")
		default:
			client.room.broadcast <- incMessage
		}
	}
}

func (client *Client) websocketWriter() {
	defer client.connection.Close()
	for {
		select {
		case msg := <-client.send:
			err := client.connection.WriteJSON(msg.ToJson())
			if err != nil {
				log.Println("error writing to client:", err)
				return
			}
		}
	}
}

func (client *Client) reset() {
	client.Guess = 0
}

func (client *Client) toJson() userDTO {
	if client.Role == Developer {
		return map[string]interface{}{
			"name":  client.Name,
			"role":  client.Role,
			"guess": client.Guess,
		}
	}
	return map[string]interface{}{
		"name": client.Name,
		"role": client.Role,
	}
}
