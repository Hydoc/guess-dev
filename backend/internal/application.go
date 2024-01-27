package internal

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"slices"
	"sort"
	"strings"
)

type Application struct {
	router   *mux.Router
	upgrader *websocket.Upgrader
	hub      *Hub
}

func (app *Application) ConfigureRouting() *mux.Router {
	app.router.HandleFunc("/api/estimation/room/{id}/product-owner", func(writer http.ResponseWriter, request *http.Request) {
		app.handleWs(app.hub, writer, request)
	}).Queries("name", "{name:.*}")
	app.router.HandleFunc("/api/estimation/room/{id}/developer", func(writer http.ResponseWriter, request *http.Request) {
		app.handleWs(app.hub, writer, request)
	}).Queries("name", "{name:.*}")
	app.router.HandleFunc("/api/estimation/room/{id}/users/exists", app.handleUserInRoomExists).Methods(http.MethodGet).Queries("name", "{name:.*}")
	app.router.HandleFunc("/api/estimation/room/{id}/users", app.handleFetchUsers).Methods(http.MethodGet)
	app.router.HandleFunc("/api/estimation/room/{id}/state", app.handleRoundInRoomInProgress).Methods(http.MethodGet)
	app.router.HandleFunc("/api/estimation/room/rooms", app.handleFetchActiveRooms).Methods(http.MethodGet)
	app.router.Use(app.contentTypeJsonMiddleware)
	return app.router
}

func (app *Application) contentTypeJsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(writer, request)
	})
}

func (app *Application) handleRoundInRoomInProgress(writer http.ResponseWriter, request *http.Request) {
	roomId := mux.Vars(request)["id"]
	json.NewEncoder(writer).Encode(map[string]bool{
		"inProgress": app.hub.IsRoundInRoomInProgress(roomId),
	})
}

func (app *Application) handleUserInRoomExists(writer http.ResponseWriter, request *http.Request) {
	roomId := mux.Vars(request)["id"]

	name := request.URL.Query().Get("name")
	if len(name) == 0 {
		writer.WriteHeader(400)
		json.NewEncoder(writer).Encode(map[string]string{
			"message": "name is missing in query",
		})
		return
	}

	for client := range app.hub.clients {
		if client.Name == name && roomId == client.RoomId {
			writer.WriteHeader(409)
			json.NewEncoder(writer).Encode(map[string]bool{
				"exists": true,
			})
			return
		}
	}
	json.NewEncoder(writer).Encode(map[string]bool{
		"exists": false,
	})
}

func (app *Application) handleFetchActiveRooms(writer http.ResponseWriter, _ *http.Request) {
	activeRooms := []string{}
	for c := range app.hub.clients {
		if !slices.Contains(activeRooms, c.RoomId) {
			activeRooms = append(activeRooms, c.RoomId)
		}
	}
	slices.Sort(activeRooms)
	json.NewEncoder(writer).Encode(activeRooms)
}

func (app *Application) handleFetchUsers(writer http.ResponseWriter, request *http.Request) {
	roomId := mux.Vars(request)["id"]

	var usersInRoom = map[string][]userDTO{
		"productOwnerList": {},
		"developerList":    {},
	}
	var clients []*Client

	for client := range app.hub.clients {
		if client.RoomId == roomId {
			clients = append(clients, client)
		}
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Name < clients[j].Name
	})

	for _, c := range clients {
		switch c.Role {
		case Developer:
			usersInRoom["developerList"] = append(usersInRoom["developerList"], c.toJson())
		case ProductOwner:
			usersInRoom["productOwnerList"] = append(usersInRoom["productOwnerList"], c.toJson())
		}
	}
	json.NewEncoder(writer).Encode(usersInRoom)
}

func (app *Application) handleWs(hub *Hub, writer http.ResponseWriter, request *http.Request) {
	roomId := mux.Vars(request)["id"]

	name := request.URL.Query().Get("name")
	if len(name) == 0 {
		writer.WriteHeader(400)
		json.NewEncoder(writer).Encode(map[string]string{
			"message": "name is missing in query",
		})
		return
	}

	connection, err := app.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	var client *Client
	if strings.Contains(request.URL.Path, "product-owner") {
		client = newProductOwner(roomId, name, hub, connection)
	} else {
		client = newDeveloper(roomId, name, hub, connection)
	}
	client.hub.register <- client
	client.hub.roomBroadcast <- newRoomBroadcast(roomId, newJoin())

	go client.websocketReader()
	go client.websocketWriter()
}

func NewApplication(router *mux.Router, upgrader *websocket.Upgrader, hub *Hub) *Application {
	return &Application{
		router:   router,
		upgrader: upgrader,
		hub:      hub,
	}
}
