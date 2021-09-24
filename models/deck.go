package models

import (
	"math/rand"
	"time"
)

var AvailableLetters = []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'K', 'L', 'M', 'N', 'O', 'P', 'R', 'S', 'T', 'U', 'W', 'Y'}

func InitializeDeck() map[byte]int {
	return map[byte]int{
		'A': 4,
		'B': 2,
		'C': 3,
		'D': 3,
		'E': 6,
		'F': 2,
		'G': 2,
		'H': 3,
		'I': 4,
		'K': 2,
		'L': 3,
		'M': 2,
		'N': 3,
		'O': 4,
		'P': 2,
		'R': 4,
		'S': 4,
		'T': 4,
		'U': 3,
		'W': 2,
		'Y': 2,
	}
}

func UpdateDeck(deck map[byte]int, word string) {
	for _, letter := range word {
		b := byte(letter)
		deck[b] = deck[b] - 1
	}
}

func CheckIfCardsAvailableForWord(deck map[byte]int, word string) bool {
	for _, letter := range word {
		b := byte(letter)
		if deck[b] == 0 {
			return false
		}
	}
	return true
}

func GetRandomCardFromDeck(deck map[byte]int) byte {
	rand.Seed(time.Now().UnixNano())
	var letter byte
	for {
		i := rand.Intn(len(AvailableLetters) - 1)
		if deck[AvailableLetters[i]] > 0 {
			deck[AvailableLetters[i]] = deck[AvailableLetters[i]] - 1
			letter = AvailableLetters[i]
			break
		}
	}
	return letter
}
