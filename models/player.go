package models

type Player struct {
	MsgChan     chan []byte
	ExitChan    chan int
	ID          int
	Username    string
	PlayerWord  string
	GuessedWord []byte
	GuessIdx    int
}

type Dummy struct {
	ID     int
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
		ID:     5,
		Letter: 'M',
	})
	deck['M'] = deck['M'] - 1
	dummies = append(dummies, &Dummy{
		ID:     6,
		Letter: 'U',
	})
	deck['U'] = deck['U'] - 1
	dummies = append(dummies, &Dummy{
		ID:     7,
		Letter: '*',
	})
	return dummies
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
