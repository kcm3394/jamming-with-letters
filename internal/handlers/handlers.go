package handlers

import (
	"encoding/csv"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/kcm3394/jamming-with-letters/models"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	wsChan  = make(chan WsPayload)
	clients = make(map[WebSocketConnection]*models.Player)
	started = false
	dummies []*models.Dummy
)

type WebSocketConnection struct {
	*websocket.Conn
}

type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

type Display struct {
	ID     string `json:"id"`
	Letter string `json:"letter"`
}

type WsJsonDisplay struct {
	Action         string    `json:"action"`
	DisplayMsg     []Display `json:"display_msg"`
	MessageType    string    `json:"message_type"`
	ConnectedUsers []string  `json:"connected_users"`
}

type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	if len(clients) == 4 || started {
		log.Println("Maxed out on players / Game in progress")
		return
	}

	log.Println("Client connected to endpoint")

	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = &models.Player{
		MsgChan:  make(chan []byte),
		ExitChan: make(chan int),
	}

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)
}

func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err == nil {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		e := <-wsChan

		switch e.Action {
		case "username":
			clients[e.Conn].Username = e.Username
			response.Action = "list_users"
			response.ConnectedUsers = getUserList()
			broadcastToAll(response)
			break
		case "left":
			response.Action = "list_users"
			delete(clients, e.Conn)
			response.ConnectedUsers = getUserList()
			broadcastToAll(response)
			break
		case "broadcast":
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			broadcastToAll(response)
			break
		case "start":
			started = true
			startGame(clients, 4) //TODO hard-coded word length
			for conn, client := range clients {
				response := getPlayerDisplay(client.ID)
				response.ConnectedUsers = getUserList()
				sendDisplayMsgToOnePlayer(response, conn)
			}
			broadcastDisplayToAll(getDummyDisplay())
			break
		}
	}
}

func getUserList() []string {
	var userList []string
	for _, x := range clients {
		if x.Username != "" {
			userList = append(userList, x.Username)
		}
	}
	sort.Strings(userList)
	return userList
}

func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func broadcastDisplayToAll(response WsJsonDisplay) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func getPlayerDisplay(currentPlayerID int) WsJsonDisplay {
	var response WsJsonDisplay
	response.Action = "player_beginning_display"
	displayMsg := make([]Display, len(clients))

	for _, client := range clients {
		if client.ID == currentPlayerID {
			displayMsg[client.ID - 1] = Display{
				ID: strconv.Itoa(client.ID),
				Letter: "?",
			}
			continue
		}
		displayMsg[client.ID - 1] = Display{
			ID:     strconv.Itoa(client.ID),
			Letter: string(client.PlayerWord[client.GuessIdx]),
		}
	}

	response.DisplayMsg = displayMsg
	return response
}

func getDummyDisplay() WsJsonDisplay {
	var response WsJsonDisplay
	response.Action = "dummy_beginning_display"
	displayMsg := make([]Display, len(dummies))

	for i, dummy := range dummies {
		displayMsg[i] = Display{
			ID:     strconv.Itoa(dummy.ID),
			Letter: string(dummy.Letter),
		}
	}

	response.DisplayMsg = displayMsg
	return response
}

func sendDisplayMsgToOnePlayer(response WsJsonDisplay, conn WebSocketConnection) {
	err := conn.WriteJSON(response)
	if err != nil {
		log.Println("websocket err")
		_ = conn.Close()
		delete(clients, conn)
	}
}

func startGame(clients map[WebSocketConnection]*models.Player, wordLength int) {
	//dictionary, err := loadDictionary()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	deck := models.InitializeDeck()
	//players := models.InitializePlayers(deck, *playerCount, dictionary)
	initializePlayers(deck, clients, wordLength)
	dummies = models.InitializeDummies(deck, len(clients))
}

func loadDictionary() ([]string, error) {
	f, err := os.Open("words/common-words.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	var dict []string

	for {
		word, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		dict = append(dict, word[0])
	}

	return dict, err
}

func initializePlayers(deck map[byte]int, players map[WebSocketConnection]*models.Player, wordLength int) {
	testWords := []string{"KEPT", "GALE", "SHOE", "HERB"}
	id := 1
	for _, player := range players {
		player.ID = id
		// player.PlayerWord = getRandomWord(dictionary, deck)
		player.PlayerWord = testWords[id-1]
		player.GuessedWord = make([]byte, wordLength)
		player.GuessIdx = 0

		fmt.Println(player.PlayerWord)
		models.UpdateDeck(deck, player.PlayerWord)
		id++
	}
}

