package models

import "fmt"

type Player struct {
	ID int
	PlayerWord string
	GuessedWord []byte
	GuessIdx int
}

type Dummy struct {
	ID int
	Letter byte
}

func InitializeDummies(deck map[byte]int, playerCount int) []*Dummy {
	//rand.Seed(time.Now().UnixNano())
	var dummies []*Dummy
	//for i := playerCount + 1; i <= 6; i++ {
	//	j := rand.Intn(len(AvailableLetters) - 1)
	//	if deck[AvailableLetters[j]] > 0 {
	//		dummies = append(dummies, &Dummy{
	//			StandNumber: CardStand{ID: i},
	//			Letter:      AvailableLetters[j],
	//		})
	//		deck[AvailableLetters[j]] = deck[AvailableLetters[j]] - 1
	//	}
	//}
	dummies = append(dummies, &Dummy{
		ID: 5,
		Letter:      'M',
	})
	deck['M'] = deck['M'] - 1
	dummies = append(dummies, &Dummy{
		ID: 6,
		Letter:      'U',
	})
	deck['U'] = deck['U'] - 1
	dummies = append(dummies, &Dummy{
		ID: 7,
		Letter:      '*',
	})
	return dummies
}

func InitializePlayers(deck map[byte]int, playerCount int, wordLength int) []*Player {
	var players []*Player
	testWords := []string{"KEPT", "GALE", "SHOE", "HERB"}
	for i := 1; i <= playerCount; i++ {
		player := &Player{
			ID: i,
			//PlayerWord:  getRandomWord(dictionary, deck),
			PlayerWord: testWords[i - 1],
			GuessedWord: make([]byte, wordLength),
			GuessIdx:    0,
		}
		fmt.Println(player.PlayerWord)
		players = append(players, player)
		UpdateDeck(deck, player.PlayerWord)
	}

	return players
}

func UpdateDummies(deck map[byte]int, dummies []*Dummy, assignments map[int][]int) {
	for _, dummy := range dummies {
		if dummy.ID == 7 { // don't update wild card
			continue
		}
		if _, ok := assignments[dummy.ID]; ok {
			dummy.Letter = GetRandomCardFromDeck(deck)
		}
	}
}
