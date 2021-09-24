package models

import "errors"

type Clue struct {
	PlayerID int
	Word     string
}

func AssignClueWord(clue *Clue, players []*Player, dummies []*Dummy) (map[int][]int, error) {
	assignments := make(map[int][]int) // player ID -> word indicies
	for i, letter := range clue.Word {
		assigned := false
		for _, player := range players {
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
