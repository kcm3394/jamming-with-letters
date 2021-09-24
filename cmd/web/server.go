package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"github.com/kcm3394/jamming-with-letters/internal/handlers"
	"github.com/kcm3394/jamming-with-letters/models"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	playerCount = flag.Int("players", 2, "how many people are playing, can be 2-6")
	wordLength  = 4
	//upgrader = websocket.Upgrader{
	//	ReadBufferSize: 1024,
	//	WriteBufferSize: 1024,
	//	CheckOrigin: func(r *http.Request) bool {
	//		return true
	//	},
	//}
)

//func wsHandler(w http.ResponseWriter, r *http.Request) {
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		log.Println(err)
//		return
//	}
//
//	log.Println("Client Connected")
//	err = conn.WriteMessage(1, []byte("Hi Client!"))
//	if err != nil {
//		log.Println(err)
//	}
//
//	reader(conn)
//}

//func reader(conn *websocket.Conn) {
//	for {
//		messageType, p, err := conn.ReadMessage()
//		if err != nil {
//			log.Println(err)
//			return
//		}
//
//		fmt.Println(string(p))
//
//		if err := conn.WriteMessage(messageType, p); err != nil {
//			log.Println(err)
//			return
//		}
//	}
//}

func main() {
	flag.Parse()
	//if *playerCount < 2 || *playerCount > 6 {
	//	fmt.Println("This game only supports 2-6 players.")
	//	return
	//}

	http.Handle("/", http.FileServer(http.Dir("./html")))
	http.HandleFunc("/ws", handlers.WsEndpoint)
	go handlers.ListenToWsChannel()

	fmt.Println("Welcome to Jamming with Letters!")
	fmt.Println("--------------------------------")
	log.Fatal(http.ListenAndServe(":8080", nil))

	//dictionary, err := loadDictionary()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	//deck := models.InitializeDeck()
	////players := models.InitializePlayers(deck, *playerCount, dictionary)
	//players :=  models.InitializePlayers(deck, *playerCount, wordLength)
	//dummies := models.InitializeDummies(deck, *playerCount)
	//
	//printRoundBeginning(*playerCount, players, dummies)
	//
	//r := bufio.NewReader(os.Stdin)
	//var err error
	//var assignments map[int][]int
	//for {
	//	clue := giveClue(r)
	//	assignments, err = models.AssignClueWord(clue, players, dummies)
	//	if err == nil {
	//		break
	//	}
	//	fmt.Println("Your clue does not match available letters. Please try again.")
	//}
	//
	//printRoundEnd(*playerCount, players, dummies, assignments)
	//
	//for _, player := range players {
	//	guessLetter(*playerCount, r, player)
	//}
	//models.UpdateDummies(deck, dummies, assignments)
	//
	//printRoundBeginning(*playerCount, players, dummies)
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

func getRandomWord(dictionary []string, deck map[byte]int) string {
	rand.Seed(time.Now().UnixNano())
	w := ""
	for len(w) != wordLength {
		i := rand.Intn(len(dictionary) - 1)
		w = dictionary[i]
	}
	if !models.CheckIfCardsAvailableForWord(deck, w) {
		getRandomWord(dictionary, deck)
	}
	return w
}

func printRoundBeginning(playerCount int, players []*models.Player, dummies []*models.Dummy) {
	for i := 1; i <= playerCount; i++ {
		fmt.Println("\nPlayer", i, "view - press Enter")
		fmt.Scanln()

		fmt.Print("Players: ")
		for _, player := range players {
			if player.ID == i {
				fmt.Print("? ")
			} else {
				fmt.Printf("%c ", player.PlayerWord[player.GuessIdx])
			}
		}

		fmt.Print("\nDummies: ")
		for _, dummy := range dummies {
			fmt.Printf("%c ", dummy.Letter)
		}
	}
}

func giveClue(r *bufio.Reader) *models.Clue {
	var clue models.Clue
	fmt.Println("\n\nTo give a clue, type player number giving the clue and the clue word in capital letters (if using the wild card, type *).")
	fmt.Println("Example: 1,BEND or 4,M**N")

	for {
		_, err := fmt.Fscanf(r, "%d,%s\n", &clue.PlayerID, &clue.Word)
		if err == nil {
			break
		}
		r.ReadBytes('\n')
		fmt.Println("Your clue must be in the correct format. Please try again.", err)
	}

	return &clue
}

func printRoundEnd(playerCount int, players []*models.Player, dummies []*models.Dummy, assignments map[int][]int) {
	for i := 1; i <= playerCount; i++ {
		fmt.Println("\nPlayer", i, "view - press Enter")
		fmt.Scanln()

		fmt.Print("Players:")
		for _, player := range players {
			if player.ID == i {
				fmt.Print(" ?")
				if val, ok := assignments[player.ID]; ok {
					for _, token := range val {
						fmt.Printf("(%d)", token)
					}
				}
			} else {
				fmt.Printf(" %c", player.PlayerWord[player.GuessIdx])
				if val, ok := assignments[player.ID]; ok {
					for _, token := range val {
						fmt.Printf("(%d)", token)
					}
				}
			}
		}

		fmt.Print("\nDummies:")
		for _, dummy := range dummies {
			fmt.Printf(" %c", dummy.Letter)
			if val, ok := assignments[dummy.ID]; ok {
				for _, token := range val {
					fmt.Printf("(%d)", token)
				}
			}
		}
	}
	fmt.Println()
}

func guessLetter(playerCount int, r *bufio.Reader, player *models.Player) {
	var input string
	fmt.Printf("\nPlayer %d: Guess your letter? Type Y/N: ", player.ID)
	fmt.Fscanf(r, "%s\n", &input)
	switch input {
	case "Y":
		for {
			err := saveGuessedLetter(r, player)
			if err == nil {
				break
			}
		}
		break
	default:
		fmt.Println("Your letter has not changed.")
	}
}

func saveGuessedLetter(r *bufio.Reader, player *models.Player) error {
	var input string
	fmt.Print("Guess: ")
	_, err := fmt.Fscanf(r, "%s\n", &input)

	if err != nil {
		fmt.Println("Invalid input. Please try again.", err)
		return err
	}

	bytes := []byte(input)
	if len(bytes) > 1 || bytes[0] < 'A' || bytes[0] > 'Z' {
		fmt.Println("Invalid input. Please try again.")
		return errors.New("invalid input")
	}

	player.GuessedWord[player.GuessIdx] = bytes[0]
	player.GuessIdx = player.GuessIdx + 1
	return nil
}
