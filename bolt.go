package main

import (
	"fmt"
	"log"

	"github.com/asdine/storm/q"

	"github.com/asdine/storm"
)

// Question objects
type Question struct {
	ID              int64 `storm:"id"`
	Rands           []int
	CurrentQuestion int
}

// InviteUser user invited object
type InviteUser struct {
	ID              int    `storm:"id,increment"`
	UserID          int    `storm:"index"`
	InvitedID       int    `storm:"unique"`
	LuckyNumber     string `storm:"index"`
	InvitedUsername string
	Username        string
	Name            string
	InvitedName     string
}

// Top user who invite most friend
type Top struct {
	ID    int `storm:"id"`
	Point int `storm:"index"`
	Name  string
}

// Score of a user
type Score struct {
	ID          int `storm:"id"`
	Score       int
	UserName    string
	FirstName   string
	LastName    string
	LuckyNumber string `storm:"index"`
}

// User for checking who
type User struct {
	ID          int
	Name        string
	LuckyNumber string
}

// QuestionStorage bot database
type QuestionStorage struct {
	db *storm.DB
}

// NewBoltStorage init storage
func NewBoltStorage() (*QuestionStorage, error) {
	db, err := storm.Open("questions.db")

	if err != nil {
		log.Printf("Cannot open db: %s", err.Error())
	}
	storage := &QuestionStorage{
		db: db,
	}
	return storage, nil
}

func (self *QuestionStorage) GetUserScore(userID int) (Score, error) {
	var result Score
	err := self.db.One("ID", userID, &result)
	if err != nil {
		log.Printf("Cannot get user score: %s", err.Error())
	}
	return result, err
}

func (self *QuestionStorage) UpdateScore(userID int, newScore Score) error {
	var score Score
	err := self.db.One("ID", userID, &score)

	if err == storm.ErrNotFound {
		log.Printf("%+v", newScore)
		err := self.db.Save(&newScore)
		if err != nil {
			log.Printf("Cannot save score: %s", err.Error())
		}
	} else {
		log.Printf("%+v", newScore)
		err := self.db.Update(&newScore)
		if err != nil {
			log.Printf("Cannot update score:  %s", err.Error())
		}
	}
	return err
}

func (self *QuestionStorage) RemoveScore(userID int) error {
	var score Score
	err := self.db.One("ID", userID, &score)

	if err == storm.ErrNotFound {
		return nil
	}
	log.Printf("%+v", score)
	err = self.db.DeleteStruct(&score)
	if err != nil {
		log.Printf("Cannot remove score: %s", err.Error())
	}
	return err
}

// Who Get list people who choose a lucky number (string)
func (self *QuestionStorage) Who(lucky string) ([]User, error) {
	result := []User{}
	var scores []Score
	var inviteUsers []InviteUser
	err := self.db.Find("LuckyNumber", lucky, &scores)
	if err != nil {
		log.Printf("Cannot find lucky number: %s", err.Error())
	}

	err = self.db.Find("LuckyNumber", lucky, &inviteUsers)
	if err != nil {
		log.Printf("Cannot find lucky number from top: %s", err.Error())
	}
	if len(inviteUsers) == 0 && len(scores) == 0 {
		self.db.AllByIndex("LuckyNumber", &scores)
		self.db.AllByIndex("LuckyNumber", &inviteUsers)
		max := User{
			ID:          scores[0].ID,
			Name:        fmt.Sprintf("%s %s", scores[0].FirstName, scores[0].LastName),
			LuckyNumber: scores[0].LuckyNumber,
		}
		n := len(scores)
		min := User{
			ID:          scores[n-1].ID,
			Name:        fmt.Sprintf("%s %s", scores[n-1].FirstName, scores[n-1].LastName),
			LuckyNumber: scores[n-1].LuckyNumber,
		}
		for _, score := range scores {
			if score.LuckyNumber > lucky && min.LuckyNumber > score.LuckyNumber {
				min = User{
					ID:          score.ID,
					Name:        fmt.Sprintf("%s %s", score.FirstName, score.LastName),
					LuckyNumber: score.LuckyNumber,
				}
			}
			if score.LuckyNumber < lucky && max.LuckyNumber < score.LuckyNumber {
				max = User{
					ID:          score.ID,
					Name:        fmt.Sprintf("%s %s", score.FirstName, score.LastName),
					LuckyNumber: score.LuckyNumber,
				}
			}
		}
		for _, score := range inviteUsers {
			if score.LuckyNumber > lucky && min.LuckyNumber > score.LuckyNumber {
				min = User{
					ID:          score.ID,
					Name:        score.Name,
					LuckyNumber: score.LuckyNumber,
				}
			}
			if score.LuckyNumber < lucky && max.LuckyNumber < score.LuckyNumber {
				max = User{
					ID:          score.ID,
					Name:        score.Name,
					LuckyNumber: score.LuckyNumber,
				}
			}
		}
		result = append(result, min)
		result = append(result, max)
	} else {
		for _, score := range scores {
			result = append(result, User{
				ID:          score.ID,
				Name:        fmt.Sprintf("%s %s", score.FirstName, score.LastName),
				LuckyNumber: score.LuckyNumber,
			})
		}
		for _, invite := range inviteUsers {
			result = append(result, User{
				ID:          invite.UserID,
				Name:        invite.Name,
				LuckyNumber: invite.LuckyNumber,
			})
		}
	}
	return result, err
}

func (self *QuestionStorage) GetCurrentQuestion(id int64) (Question, error) {
	var question Question
	log.Printf("%d", id)
	err := self.db.One("ID", id, &question)
	if err != nil {
		log.Printf("Cannot get question: %s", err.Error())
	}
	return question, err
}

func (self *QuestionStorage) UpdateQuestion(id int64, question Question) error {
	_, err := self.GetCurrentQuestion(id)
	if err != nil {
		log.Printf("Cannot get current question: %s", err.Error())
		err = self.db.Save(&question)
	} else {
		log.Printf("What question: %+v", question)
		err = self.db.Update(&question)
		log.Printf("Error: %s", err)
	}
	currentQuestion, _ := self.GetCurrentQuestion(id)
	log.Printf("current question: %+v", currentQuestion)
	return err
}

func (self *QuestionStorage) RemoveQuestion(id int64) error {
	current, err := self.GetCurrentQuestion(id)
	if err != nil {
		log.Printf("Cannot get question.")
		return err
	}
	log.Printf("Remove question: %+v", current)
	err = self.db.DeleteStruct(&current)
	if err != nil {
		log.Printf("Cannot remove question: %s", err.Error())
	}
	return err
}

func (self *QuestionStorage) GetInvitedUserWithoutLuckyNumber(userID int) (InviteUser, error) {
	var invitedUser InviteUser
	query := self.db.Select(q.And(q.Eq("UserID", userID), q.Eq("LuckyNumber", "")))
	err := query.First(&invitedUser)
	if err != nil {
		log.Printf("Cannot get user: %s", err.Error())
	}
	return invitedUser, err
}

func (self *QuestionStorage) InvitedUser(userID int, invitedUser InviteUser) error {
	err := self.db.Save(&invitedUser)
	if err != nil {
		log.Printf("Cannot add invited user: %s", err.Error())
	}
	return err
}

func (self *QuestionStorage) UpdateInviteUser(invitedUser InviteUser) error {
	err := self.db.Update(&invitedUser)
	if err != nil {
		log.Printf("Cannot update invited user: %s", err.Error())
	}
	return err
}

func (self *QuestionStorage) RemoveUser(userLeftID int) error {
	var userLeft InviteUser
	err := self.db.One("InvitedID", userLeftID, &userLeft)
	if err != nil {
		log.Printf("Cannot get user left: %s", err.Error())
		return err
	}
	self.db.DeleteStruct(&userLeft)
	return err
}

func (self *QuestionStorage) GetInvitedUser(userID int) ([]InviteUser, error) {
	var users []InviteUser
	err := self.db.Find("UserID", userID, &users)
	if err != nil {
		log.Printf("Cannot get invited user: %s", err.Error())
	}
	return users, err
}

func (self *QuestionStorage) GetInvitedUserByInvitedID(invitedID int) (InviteUser, error) {
	var user InviteUser
	err := self.db.One("InvitedID", invitedID, &user)
	if err != nil {
		log.Printf("Cannot get invited user")
	}
	return user, err
}

func (self *QuestionStorage) UpdateTop(userID int, username string, point int) error {
	var top Top
	err := self.db.One("ID", userID, &top)
	if err != nil {
		top.ID = userID
		top.Point += point
		top.Name = username
		err = self.db.Save(&top)
		if err != nil {
			log.Printf("Cannot save top point: %s", err.Error())
		} else {
			log.Printf("Saved top: %+v", top)
		}
	} else {
		point := top.Point + point
		err = self.db.UpdateField(&top, "Point", point)
		if err != nil {
			log.Printf("Cannot update top point: %s", err.Error())
		} else {
			log.Printf("Updated top: %+v", top)
		}
	}
	return err
}

func (self *QuestionStorage) GetTop() ([]Top, error) {
	var tops []Top
	err := self.db.AllByIndex("Point", &tops)
	return tops, err
}
