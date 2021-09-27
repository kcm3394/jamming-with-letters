package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/kcm3394/jamming-with-letters/models"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	wsChan           = make(chan WsPayload)
	clients          = make(map[WebSocketConnection]*models.Player)
	started          = false
	dummies          []*models.Dummy
	playersSubmitted = 0
	deck             map[byte]int
	wordLength       = 3 //TODO hard-coded word length
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
	Token  string `json:"token"`
}

type WsJsonDisplay struct {
	Action         string    `json:"action"`
	DisplayMsg     []Display `json:"display_msg"`
	MessageType    string    `json:"message_type"`
	ConnectedUsers []string  `json:"connected_users"`
}

type EndGameDisplay struct {
	ID          string `json:"id"`
	PlayerWord  string `json:"player_word"`
	GuessedWord string `json:"guessed_word"`
}

type WsJsonEndGame struct {
	Action         string           `json:"action"`
	DisplayMsg     []EndGameDisplay `json:"display_msg"`
	MessageType    string           `json:"message_type"`
	ConnectedUsers []string         `json:"connected_users"`
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
	for {
		e := <-wsChan

		switch e.Action {
		case "username":
			var response WsJsonResponse
			clients[e.Conn].Username = e.Username
			response.Action = "list_users"
			response.ConnectedUsers = getUserList()
			broadcastToAll(response)
			break
		case "left":
			var response WsJsonResponse
			response.Action = "list_users"
			delete(clients, e.Conn)
			response.ConnectedUsers = getUserList()
			broadcastToAll(response)
			break
		case "broadcast":
			var response WsJsonResponse
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			broadcastToAll(response)
			break
		case "start":
			started = true
			startGame(clients, wordLength)
			displayCardsAndTokens(nil)
			break
		case "clue":
			var response WsJsonResponse
			log.Println(e.Message)
			clue := models.Clue{
				PlayerID: clients[e.Conn].ID,
				Word:     strings.ToUpper(e.Message),
			}
			assignments, err := assignClueWord(&clue)
			if err != nil {
				response.Action = "error"
				response.Message = "Your clue does not match available letters. Please try again."
				log.Println("Your clue does not match available letters. Please try again.")
				broadcastToAll(response)
				break
			}

			displayCardsAndTokens(assignments)
			broadcastToAll(WsJsonResponse{
				Action: "disable-clue",
			})

			//prepare for next round, should not affect current display if called after displayCardsAndTokens
			models.UpdateDummies(deck, dummies, assignments)
			break
		case "letter":
			var response WsJsonResponse
			log.Println(e.Message, clients[e.Conn])
			if len(e.Message) != 1 && e.Message != "skip" {
				response.Action = "error"
				response.Message = "Your letter is not a single letter or the word 'skip'. Please try again."
				sendMsgToOnePlayer(response, e.Conn)
				break
			}

			if e.Message != "skip" {
				saveGuessedLetter(strings.ToUpper(e.Message), clients[e.Conn])
			}

			playersSubmitted++
			if playersSubmitted == len(clients) {
				// start new round
				playersSubmitted = 0
				// check end game condition
				if isGameEnd() {
					response := getEndGameDisplay()
					broadcastEndGameDisplayToAll(response)
					break
				}
				displayCardsAndTokens(nil)
				broadcastToAll(WsJsonResponse{
					Action: "disable-guess",
				})
			}
			break
		}
	}
}

func getEndGameDisplay() WsJsonEndGame {
	response := WsJsonEndGame{
		Action:     "display-end-game",
		DisplayMsg: make([]EndGameDisplay, len(clients)),
	}

	for _, client := range clients {
		playerDisplay := EndGameDisplay{
			ID:          strconv.Itoa(client.ID),
			PlayerWord:  client.PlayerWord,
			GuessedWord: string(client.GuessedWord),
		}
		response.DisplayMsg[client.ID-1] = playerDisplay
	}

	return response
}

func isGameEnd() bool {
	for _, client := range clients {
		if client.GuessIdx < wordLength {
			return false
		}
	}
	return true
}

func displayCardsAndTokens(assignments map[int][]int) {
	for conn, client := range clients {
		response := getPlayerDisplay(client.ID, assignments)
		response.ConnectedUsers = getUserList()
		sendDisplayMsgToOnePlayer(response, conn)
	}
	broadcastDisplayToAll(getDummyDisplay(assignments))
}

func saveGuessedLetter(letter string, player *models.Player) {
	if player.GuessIdx == wordLength {
		// player has already guessed all letters
		player.BonusLetter = models.GetRandomCardFromDeck(deck)
		log.Printf("New bonus letter assigned to player %d", player.ID)
		return
	}

	player.GuessedWord[player.GuessIdx] = letter[0]
	player.GuessIdx = player.GuessIdx + 1

	// player just finished guessing their word
	if player.GuessIdx == wordLength {
		player.BonusLetter = models.GetRandomCardFromDeck(deck)
		log.Printf("New bonus letter assigned to player %d", player.ID)
		return
	}

	log.Printf("Player %d guessed - new idx is %d. Guessed word so far is %s", player.ID, player.GuessIdx, player.GuessedWord)
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

func broadcastEndGameDisplayToAll(response WsJsonEndGame) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func getPlayerDisplay(currentPlayerID int, assignments map[int][]int) WsJsonDisplay {
	var response WsJsonDisplay
	response.Action = "player_display"
	displayMsg := make([]Display, len(clients))

	for _, client := range clients {
		// if player on bonus letter
		if client.GuessIdx >= wordLength {
			displayMsg[client.ID-1] = Display{
				ID:     strconv.Itoa(client.ID),
				Letter: string(client.BonusLetter),
			}
		} else {
			displayMsg[client.ID-1] = Display{
				ID:     strconv.Itoa(client.ID),
				Letter: string(client.PlayerWord[client.GuessIdx]),
			}
		}

		if assignments != nil {
			if val, ok := assignments[client.ID]; ok {
				sb := strings.Builder{}
				for _, token := range val {
					sb.Write([]byte(fmt.Sprintf("(%d)", token)))
				}
				displayMsg[client.ID-1].Token = sb.String()
			}
		}
		if client.ID == currentPlayerID {
			displayMsg[client.ID-1].Letter = "?"
		}
	}

	response.DisplayMsg = displayMsg
	return response
}

func getDummyDisplay(assignments map[int][]int) WsJsonDisplay {
	var response WsJsonDisplay
	response.Action = "dummy_display"
	displayMsg := make([]Display, len(dummies))

	for i, dummy := range dummies {
		displayMsg[i] = Display{
			ID:     strconv.Itoa(dummy.ID),
			Letter: string(dummy.Letter),
		}
		if assignments != nil {
			if val, ok := assignments[dummy.ID]; ok {
				sb := strings.Builder{}
				for _, token := range val {
					sb.Write([]byte(fmt.Sprintf("(%d)", token)))
				}
				displayMsg[i].Token = sb.String()
			}
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

func sendMsgToOnePlayer(response WsJsonResponse, conn WebSocketConnection) {
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

	deck = models.InitializeDeck()
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
	testWords := []string{"CAT", "BOT"}
	id := 1
	for _, player := range players {
		player.ID = id
		// player.PlayerWord = getRandomWord(dictionary, deck)
		player.PlayerWord = testWords[id-1]
		player.GuessedWord = make([]byte, wordLength)
		player.GuessIdx = 0
		player.BonusLetter = 0

		fmt.Println(player.PlayerWord)
		models.UpdateDeck(deck, player.PlayerWord)
		id++
	}
}

func assignClueWord(clue *models.Clue) (map[int][]int, error) {
	assignments := make(map[int][]int) // player ID -> word indicies
	for i, letter := range clue.Word {
		assigned := false
		for _, player := range clients {
			if player.ID == clue.PlayerID {
				continue
			}
			if byte(letter) == player.PlayerWord[player.GuessIdx] {
				if val, ok := assignments[player.ID]; ok { // if player letter already given word idx token
					newValue := append(val, i+1)
					assignments[player.ID] = newValue
				} else {
					assignments[player.ID] = []int{i + 1} // give player letter its first idx token
				}
				assigned = true
				break
			}
		}

		if assigned {
			continue
		}

		for _, dummy := range dummies {
			if byte(letter) == dummy.Letter {
				if val, ok := assignments[dummy.ID]; ok { // if dummy letter already given word idx token
					newValue := append(val, i+1)
					assignments[dummy.ID] = newValue
				} else {
					assignments[dummy.ID] = []int{i + 1} // give dummy letter its first idx token
				}
				assigned = true
				break
			}
		}

		if !assigned {
			return nil, errors.New("word does not match available letters")
		}
	}

	return assignments, nil
}
