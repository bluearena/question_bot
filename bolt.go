package main

import (
	"log"

	"github.com/asdine/storm/q"

	"github.com/asdine/storm"
)

type Question struct {
	ID              int64 `storm:"id"`
	Rands           []int
	CurrentQuestion int
}

type InviteUser struct {
	ID              int    `storm:"id,increment"`
	UserID          int    `storm:"index"`
	InvitedID       int    `storm:"unique"`
	LuckyNumber     string `storm:"index"`
	InvitedUsername string
	Username        string
}

type Score struct {
	ID          int `storm:"id"`
	Score       int
	UserName    string
	FirstName   string
	LastName    string
	LuckyNumber string `storm:"index"`
}

type QuestionStorage struct {
	db *storm.DB
}

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

func (self *QuestionStorage) Who(lucky string) ([]Score, error) {
	var scores []Score
	log.Printf("Lucky number: %s", lucky)
	err := self.db.Find("LuckyNumber", lucky, &scores)
	log.Printf("%+v", scores)
	if err != nil {
		log.Printf("Cannot find lucky users: %s", err.Error())
	}
	return scores, err
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
		log.Printf("Cannot remove question: %s", err.Error)
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
