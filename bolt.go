package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
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
	Valid           bool
}

// Top user who invite most friend
type Top struct {
	ID    int `storm:"id"`
	Point int `storm:"index"`
	Name  string
	Valid bool
}

// Score of a user
type Score struct {
	ID          int `storm:"id"`
	Score       int
	UserName    string
	FirstName   string
	LastName    string
	LuckyNumber string `storm:"index"`
	Valid       bool
}

// User for checking who
type User struct {
	ID          int
	Name        string
	LuckyNumber string
}

// NewUser return a new user
func NewUser(id int, name, lucky string) User {
	return User{
		ID:          id,
		Name:        name,
		LuckyNumber: lucky,
	}
}

// QuestionStorage bot database
type QuestionStorage struct {
	db *storm.DB
}

// NewBoltStorage init storage
func NewBoltStorage() (*QuestionStorage, error) {
	db, err := storm.Open("/db/questions.db")

	if err != nil {
		log.Printf("Cannot open db: %s", err.Error())
	}
	storage := &QuestionStorage{
		db: db,
	}
	return storage, nil
}

// GetUserScore get user score
func (storage *QuestionStorage) GetUserScore(userID int) (Score, error) {
	var result Score
	err := storage.db.One("ID", userID, &result)
	if err != nil {
		log.Printf("Cannot get user %d score: %s", userID, err.Error())
	}
	return result, err
}

// UpdateScore update user score
func (storage *QuestionStorage) UpdateScore(userID int, newScore Score) error {
	var score Score
	err := storage.db.One("ID", userID, &score)

	if err == storm.ErrNotFound {
		log.Printf("%+v", newScore)
		err := storage.db.Save(&newScore)
		if err != nil {
			log.Printf("Cannot save score: %s", err.Error())
		}
	} else {
		log.Printf("%+v", newScore)
		err := storage.db.Update(&newScore)
		if err != nil {
			log.Printf("Cannot update score:  %s", err.Error())
		}
		if newScore.Valid == false {
			err = storage.db.UpdateField(&newScore, "Valid", false)
			if err != nil {
				log.Printf("Cannot update score valid:  %s", err.Error())
			}
		}
	}
	return err
}

// RemoveScore remove user score from db
func (storage *QuestionStorage) RemoveScore(userID int) error {
	var score Score
	err := storage.db.One("ID", userID, &score)

	if err == storm.ErrNotFound {
		return nil
	}
	log.Printf("%+v", score)
	err = storage.db.DeleteStruct(&score)
	if err != nil {
		log.Printf("Cannot remove score: %s", err.Error())
	}
	return err
}

//Abs return absolute value of a number
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Who Get list people who choose a lucky number (string)
func (storage *QuestionStorage) Who(lucky string) ([]User, error) {
	result := []User{}
	var scores []Score
	var inviteUsers []InviteUser
	err := storage.db.Find("LuckyNumber", lucky, &scores)
	if err != nil {
		log.Printf("Cannot find lucky number: %s", err.Error())
		return result, err
	}

	err = storage.db.Find("LuckyNumber", lucky, &inviteUsers)
	if err != nil {
		log.Printf("Cannot find lucky number from top: %s", err.Error())
		return result, err
	}
	for _, score := range scores {
		if score.Valid == false {
			continue
		}
		user := NewUser(score.ID, fmt.Sprintf("%s %s", score.FirstName, score.LastName), score.LuckyNumber)
		result = append(result, user)
	}
	for _, invite := range inviteUsers {
		if invite.Valid == false {
			continue
		}
		user := NewUser(invite.UserID, invite.Name, invite.LuckyNumber)
		result = append(result, user)
	}
	if len(result) != 0 {
		return result, err
	}
	storage.db.AllByIndex("LuckyNumber", &scores)
	storage.db.AllByIndex("LuckyNumber", &inviteUsers)
	max := NewUser(0, "", "0000")
	min := NewUser(0, "", "9999")
	for _, score := range scores {
		if score.Valid {
			if score.LuckyNumber > lucky && min.LuckyNumber > score.LuckyNumber {
				min = NewUser(score.ID, fmt.Sprintf("%s %s", score.FirstName, score.LastName), score.LuckyNumber)
			}
			if score.LuckyNumber < lucky && max.LuckyNumber < score.LuckyNumber {
				max = NewUser(score.ID, fmt.Sprintf("%s %s", score.FirstName, score.LastName), score.LuckyNumber)
			}
		}
	}
	for _, score := range inviteUsers {
		if score.Valid {
			if score.LuckyNumber > lucky && min.LuckyNumber > score.LuckyNumber {
				min = NewUser(score.UserID, score.Name, score.LuckyNumber)
			}
			if score.LuckyNumber < lucky && max.LuckyNumber < score.LuckyNumber {
				max = NewUser(score.UserID, score.Name, score.LuckyNumber)
			}
		}
	}
	if min.ID != 0 {
		if max.ID == 0 {
			result = append(result, min)
		} else {
			intMin, _ := strconv.Atoi(min.LuckyNumber)
			intMax, _ := strconv.Atoi(max.LuckyNumber)
			intLucky, _ := strconv.Atoi(lucky)
			if Abs(intMin-intLucky) < Abs(intMax-intLucky) {
				result = append(result, min)
			} else {
				result = append(result, max)
			}
		}
	} else if max.ID != 0 {
		result = append(result, max)
	}
	return result, err
}

// GetCurrentQuestion get current question for user
func (storage *QuestionStorage) GetCurrentQuestion(id int64) (Question, error) {
	var question Question
	err := storage.db.One("ID", id, &question)
	if err != nil {
		log.Printf("Cannot get question: %s", err.Error())
		return question, err
	}
	return question, err
}

// UpdateQuestion update current question
func (storage *QuestionStorage) UpdateQuestion(id int64, question Question) error {
	_, err := storage.GetCurrentQuestion(id)
	if err != nil {
		log.Printf("Cannot get current question: %s", err.Error())
		err = storage.db.Save(&question)
	} else {
		err = storage.db.Update(&question)
		log.Printf("Error: %s", err)
	}
	return err
}

// RemoveQuestion remove question list to restart
func (storage *QuestionStorage) RemoveQuestion(id int64) error {
	current, err := storage.GetCurrentQuestion(id)
	if err != nil {
		log.Printf("Cannot get question.")
		return err
	}
	log.Printf("Remove question: %+v", current)
	err = storage.db.DeleteStruct(&current)
	if err != nil {
		log.Printf("Cannot remove question: %s", err.Error())
	}
	return err
}

// GetInvitedUserWithoutLuckyNumber get user list so the bot can update lucky number for that user
func (storage *QuestionStorage) GetInvitedUserWithoutLuckyNumber(userID int) ([]InviteUser, error) {
	var invitedUsers []InviteUser
	query := storage.db.Select(q.And(q.Eq("UserID", userID), q.Eq("LuckyNumber", "")))
	err := query.Find(&invitedUsers)
	if err != nil {
		log.Printf("Cannot get user: %s", err.Error())
	}
	return invitedUsers, err
}

// InvitedUser get invited user
func (storage *QuestionStorage) InvitedUser(userID int, invitedUser InviteUser) error {
	err := storage.db.Save(&invitedUser)
	if err != nil {
		log.Printf("Cannot add invited user: %s", err.Error())
	}
	return err
}

// UpdateInviteUser update lucky number to invite user
func (storage *QuestionStorage) UpdateInviteUser(invitedUser InviteUser) error {
	err := storage.db.Update(&invitedUser)
	if err != nil {
		log.Printf("Cannot update invited user: %s", err.Error())
	}
	if invitedUser.Valid == false {
		err = storage.db.UpdateField(&invitedUser, "Valid", false)
		if err != nil {
			log.Printf("Cannot update invited valid: %s", err.Error())
		}
	}
	return err
}

// RemoveUser remove a user
func (storage *QuestionStorage) RemoveUser(userLeftID int) error {
	var userLeft InviteUser
	err := storage.db.One("InvitedID", userLeftID, &userLeft)
	if err != nil {
		log.Printf("Cannot get user left: %s", err.Error())
		return err
	}
	storage.db.DeleteStruct(&userLeft)
	return err
}

// GetInvitedUser Get a user
func (storage *QuestionStorage) GetInvitedUser(userID int) ([]InviteUser, error) {
	var users []InviteUser
	err := storage.db.Find("UserID", userID, &users)
	if err != nil {
		log.Printf("Cannot get invited user: %s", err.Error())
	}
	return users, err
}

// GetInvitedUserByInvitedID get invited user by invited id
func (storage *QuestionStorage) GetInvitedUserByInvitedID(invitedID int) (InviteUser, error) {
	var user InviteUser
	err := storage.db.One("InvitedID", invitedID, &user)
	if err != nil {
		log.Printf("Cannot get invited user")
	}
	return user, err
}

// UpdateTop update top point
func (storage *QuestionStorage) UpdateTop(userID int, username string, point int) error {
	var top Top
	err := storage.db.One("ID", userID, &top)
	if err != nil {
		top.ID = userID
		top.Point += point
		top.Name = username
		top.Valid = true
		err = storage.db.Save(&top)
		if err != nil {
			log.Printf("Cannot save top point: %s", err.Error())
		} else {
			log.Printf("Saved top: %+v", top)
		}
	} else {
		point := top.Point + point
		err = storage.db.UpdateField(&top, "Point", point)
		if err != nil {
			log.Printf("Cannot update top point: %s", err.Error())
		} else {
			log.Printf("Updated top: %+v", top)
		}
	}
	return err
}

// GetTop get top point
func (storage *QuestionStorage) GetTop() ([]Top, error) {
	var tops []Top
	err := storage.db.AllByIndex("Point", &tops)
	return tops, err
}

// GetTopByUserID get top by user id
func (storage *QuestionStorage) GetTopByUserID(userID int) (Top, error) {
	var top Top
	err := storage.db.One("ID", userID, &top)
	return top, err
}

// UpdateTopObject update top object
func (storage *QuestionStorage) UpdateTopObject(top Top) error {
	err := storage.db.Update(&top)
	if err != nil {
		log.Printf("Cannot update top object: %s", err.Error())
	}
	if top.Valid == false {
		err = storage.db.UpdateField(&top, "Valid", false)
		if err != nil {
			log.Printf("Cannot update top valid:  %s", err.Error())
		}
	}
	return err
}

// GetAllInvitedUser Get all invited users
func (storage *QuestionStorage) GetAllInvitedUser() ([]InviteUser, error) {
	var users []InviteUser
	err := storage.db.All(&users)
	if err != nil {
		log.Printf("Cannot get all invited user: %s", err.Error())
	}
	return users, err
}

// GetAllUserScore Get all user score
func (storage *QuestionStorage) GetAllUserScore() ([]Score, error) {
	var scores []Score
	err := storage.db.All(&scores)
	if err != nil {
		log.Printf("Cannot get all scores: %s", err.Error())
	}
	return scores, err
}
