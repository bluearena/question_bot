package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type BotConfig struct {
	Key string `json:"bot_key"`
}

type Bot struct {
	bot     *tb.Bot
	storage *QuestionStorage
}

type Questions []struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   int      `json:"answer"`
}

const CHAT_GROUP string = "testquestion1234"

// map[chatID]questionID
var questions Questions
var lucky map[string]string
var replyKeysTwo [][]tb.ReplyButton
var replyKeysFour [][]tb.ReplyButton

func readConfigFromFile(path string) (BotConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return BotConfig{}, err
	} else {
		result := BotConfig{}
		err := json.Unmarshal(data, &result)
		return result, err
	}
}

func readQuestionsFromFile(path string) (Questions, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return Questions{}, err
	} else {
		result := Questions{}
		err := json.Unmarshal(data, &result)
		return result, err
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	path := "./config.json"
	botConfig, err := readConfigFromFile(path)
	if err != nil {
		log.Fatal(err)
	}
	tbot, err := tb.NewBot(tb.Settings{
		Token:  botConfig.Key,
		Poller: &tb.LongPoller{Timeout: 5 * time.Second},
	})
	if err != nil {
		log.Fatalf("Cannot initiate new bot: %s", err.Error())
	}
	storage, err := NewBoltStorage()
	mybot := Bot{
		bot:     tbot,
		storage: storage,
	}
	if err != nil {
		log.Panic(err)
		return
	}

	questionPath := "./questions.json"
	questions, err = readQuestionsFromFile(questionPath)
	if err != nil {
		log.Fatal(err)
	}

	replyKeysFour = mybot.initReplyKeys([]string{"A", "B", "C", "D"})
	replyKeysTwo = mybot.initReplyKeys([]string{"A", "B"})

	mybot.bot.Handle("/start", func(m *tb.Message) {
		mybot.handleStart(m)
	})

	mybot.bot.Handle("/A", func(m *tb.Message) {
		mybot.handleAnswer(m, 0)
	})
	mybot.bot.Handle("/B", func(m *tb.Message) {
		mybot.handleAnswer(m, 1)
	})
	mybot.bot.Handle("/C", func(m *tb.Message) {
		mybot.handleAnswer(m, 2)
	})
	mybot.bot.Handle("/D", func(m *tb.Message) {
		mybot.handleAnswer(m, 3)
	})

	mybot.bot.Handle("/who", func(m *tb.Message) {
		mybot.handleWho(m)
	})

	mybot.bot.Handle("/me", func(m *tb.Message) {
		mybot.handleMe(m)
	})

	mybot.bot.Handle("/add", func(m *tb.Message) {
		mybot.handleAdd(m)
	})

	mybot.bot.Handle(tb.OnText, func(m *tb.Message) {
		mybot.handleText(m)
	})

	mybot.bot.Handle(tb.OnUserJoined, func(m *tb.Message) {
		log.Printf("User joined: %+v", m.Chat)
		mybot.handleUserJoined(m)
	})

	mybot.bot.Handle(tb.OnUserLeft, func(m *tb.Message) {
		log.Printf("User left: %+v", m)
		mybot.handleUserLeft(m)
	})

	mybot.bot.Start()
}

func updateCurrentCommand(command string, m *tb.Message) {
	if len(lucky) == 0 {
		lucky = map[string]string{}
	}
	lucky[fmt.Sprintf("%s_%s", m.Chat.ID, m.Sender.ID)] = command
}

func (self Bot) handleUserJoined(m *tb.Message) {
	if m.Sender.ID == m.UserJoined.ID || m.Chat.Username != CHAT_GROUP {
		return
	}
	message := fmt.Sprintf("Bạn đã add @%s vào group @%s. Bạn được thêm 1 lần chọn số may mắn. Bạn có thể /add để thêm số may mắn.", m.UserJoined.Username, m.Chat.Username)
	self.bot.Send(m.Sender, message)
	inviteUser := InviteUser{
		UserID:          m.Sender.ID,
		InvitedID:       m.UserJoined.ID,
		LuckyNumber:     "",
		Username:        m.Sender.Username,
		InvitedUsername: m.UserJoined.Username,
	}
	self.storage.InvitedUser(m.Sender.ID, inviteUser)
	updateCurrentCommand("invited", m)
}

func (self Bot) handleMe(m *tb.Message) {
	if !m.Private() {
		self.bot.Reply(m, "Chúng tôi sẽ trả lời riêng cho bạn.")
	}
	score, _ := self.storage.GetUserScore(m.Sender.ID)
	message := fmt.Sprintf("Trả lời câu hỏi: %d/5, số may mắn: %s\n", score.Score, score.LuckyNumber)
	invites, err := self.storage.GetInvitedUser(m.Sender.ID)
	if err != nil && err.Error() == "not found" {
		message += fmt.Sprintf("Bạn chưa mời thêm người bạn nào vào %s. \n", CHAT_GROUP)
	} else {
		message += fmt.Sprintf("Bạn đã mời: \n")
		for _, user := range invites {
			message += fmt.Sprintf("@%s, số may mắn: %s \n", user.InvitedUsername, user.LuckyNumber)
		}
	}
	_, err = self.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		message += fmt.Sprintf("Bạn có thể /add để thêm số may mắn.")
	}
	self.bot.Send(m.Sender, message)
}

func (self Bot) handleAdd(m *tb.Message) {
	if !m.Private() {
		self.bot.Reply(m, "Chúng tôi sẽ trả lời riêng cho bạn.")
	}
	_, err := self.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		updateCurrentCommand("invited", m)
		self.bot.Send(m.Sender, "Điền 4 chữ số may mắn: ")
	} else {
		self.bot.Send(m.Sender, "Bạn không còn lượt chọn số may mắn nào.")
	}
}

func (self Bot) handleUserLeft(m *tb.Message) {
	if m.Chat.Username != CHAT_GROUP {
		return
	}
	exist, err := self.storage.GetInvitedUserByInvitedID(m.UserLeft.ID)
	if err == nil {
		self.storage.RemoveUser(m.UserLeft.ID)
		message := fmt.Sprintf("@%s đã rời khỏi group @%s. Số may mắn bạn chọn cho @%s đã bị hủy.", exist.InvitedUsername, CHAT_GROUP, exist.InvitedUsername)
		user := tb.User{
			ID: exist.UserID,
		}
		self.bot.Send(&user, message)
	}
}

func (self Bot) initReplyKeys(questionOptions []string) [][]tb.ReplyButton {
	replyKeys := [][]tb.ReplyButton{}
	replyKeyOne := []tb.ReplyButton{}
	for key, _ := range questionOptions {
		replyBtn := tb.ReplyButton{Text: questionOptions[key]}
		self.bot.Handle(&replyBtn, func(m *tb.Message) {
			option := 0
			for i, v := range questionOptions {
				if v == replyBtn.Text {
					option = i
				}
			}
			self.handleAnswer(m, option)
		})
		replyKeyOne = append(replyKeyOne, replyBtn)
		if key%2 == 1 {
			replyKeys = append(replyKeys, replyKeyOne)
			replyKeyOne = []tb.ReplyButton{}
		}
	}
	return replyKeys
}

func (self Bot) handleText(m *tb.Message) {
	switch lucky[fmt.Sprintf("%s_%s", m.Chat.ID, m.Sender.ID)] {
	case "lucky":
		self.handleUpateLucky(m)
	case "who":
		self.handleCheckWho(m, m.Text)
	case "invited":
		self.handleInvited(m)
	}
}

func (self Bot) handleInvited(m *tb.Message) {
	text := strings.TrimSpace(m.Text)
	log.Printf("Invited: %s", text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		self.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số."))
	} else {
		invitedUser, err := self.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
		if err != nil {
			log.Printf("Cannot get score: %s", err.Error())
		}
		invitedUser.LuckyNumber = text
		err = self.storage.UpdateInviteUser(invitedUser)
		if err != nil {
			log.Printf("Cannot update lucky number: %s", err.Error())
		}
		self.bot.Send(m.Chat, fmt.Sprintf("Con số may mắn bạn đã chọn là: %s, chúng tôi sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc.", text))
		updateCurrentCommand("", m)
	}
}

func (self Bot) handleUpateLucky(m *tb.Message) {
	text := strings.TrimSpace(m.Text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		self.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số."))
	} else {
		score, err := self.storage.GetUserScore(m.Sender.ID)
		if err != nil {
			log.Printf("Cannot get score: %s", err.Error())
		}
		score.LuckyNumber = text
		err = self.storage.UpdateScore(m.Sender.ID, score)
		if err != nil {
			log.Printf("Cannot update lucky number: %s", err.Error())
		}
		score, _ = self.storage.GetUserScore(m.Sender.ID)
		self.bot.Send(m.Chat, fmt.Sprintf("Con số may mắn bạn đã chọn là: %s, chúng tôi sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc.", score.LuckyNumber))
		updateCurrentCommand("", m)
	}
}

func (self Bot) handleCheckWho(m *tb.Message, luckyNumber string) {
	luckyStr := strings.TrimSpace(luckyNumber)
	matched, err := regexp.MatchString(`^\d{4,4}$`, luckyStr)
	log.Printf("%+v", matched)
	if err != nil {
		log.Printf("Cannot match lucky string: %s", err.Error())
	}
	if !matched {
		self.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số để kiểm tra người may mắn."))
	} else {
		updateCurrentCommand("", m)
		users, err := self.storage.Who(luckyStr)
		if err != nil && err.Error() != "not found" {
			log.Printf("Cannot get user: %s", err)
		}
		if err != nil && err.Error() == "not found" {
			self.bot.Reply(m, fmt.Sprintf("Chưa có người dùng nào chọn số %s.", luckyStr))
			return
		}
		message := fmt.Sprintf("Danh sách những người đã chọn số %s: \n\n", luckyStr)
		for _, user := range users {
			message += fmt.Sprintf("@%s \n", user.UserName)
		}
		self.bot.Reply(m, message, &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	}
}

// send the next question
func (self Bot) next(m *tb.Message) {
	currentQuestion, _ := self.storage.GetCurrentQuestion(m.Chat.ID)
	nextQuestion := currentQuestion.CurrentQuestion
	rands := currentQuestion.Rands

	if nextQuestion+1 > len(rands) {
		self.finish(m)
	} else {
		question := questions[rands[nextQuestion]]

		message := fmt.Sprintf("%d. %s \n\n", nextQuestion+1, question.Question)
		questionOptions := []string{"A", "B", "C", "D"}
		for key, option := range question.Options {
			message += fmt.Sprintf("/%s %s \n", questionOptions[key], option)
		}
		replyKeys := replyKeysFour
		if len(question.Options) == 2 {
			replyKeys = replyKeysTwo
		}
		self.bot.Send(m.Chat, message, &tb.SendOptions{
			DisableWebPagePreview: true,
			ParseMode:             tb.ModeMarkdown,
			ReplyMarkup: &tb.ReplyMarkup{
				ReplyKeyboard:       replyKeys,
				ResizeReplyKeyboard: true,
				OneTimeKeyboard:     true,
			},
		})
	}
}

func (self Bot) finish(m *tb.Message) {
	score, _ := self.storage.GetUserScore(m.Sender.ID)
	message := fmt.Sprintf("Bạn đã hoàn thành. Bạn trả lời đúng: %d/5 câu hỏi.\n", score.Score)
	if score.Score == 5 {
		message += fmt.Sprintf("Nhập 1 số có 5 chữ số để quay số may mắn.")
		updateCurrentCommand("lucky", m)
	} else {
		message += fmt.Sprintf("Rất tiếc bạn chưa đủ điều kiện quay số may mắn. Thử lại để đạt mức điểm cao hơn: /start")
	}

	self.bot.Send(m.Chat, message,
		&tb.SendOptions{
			ReplyMarkup: &tb.ReplyMarkup{
				ReplyKeyboardRemove: true,
			},
		})
}

func (self Bot) handleAnswer(m *tb.Message, option int) {
	log.Printf("%d", option)
	currentQuestion, _ := self.storage.GetCurrentQuestion(m.Chat.ID)
	log.Printf("%+v", currentQuestion)
	current := questions[currentQuestion.Rands[currentQuestion.CurrentQuestion]]
	log.Printf("%+v", current)
	if option+1 > len(current.Options) {
		self.bot.Send(m.Chat, fmt.Sprintf("Câu hỏi không có phương án bạn chọn."))
		return
	}

	score, err := self.storage.GetUserScore(m.Sender.ID)
	if err != nil && err.Error() == "not found" {
		score = Score{
			ID:        m.Sender.ID,
			Score:     0,
			UserName:  m.Sender.Username,
			FirstName: m.Sender.FirstName,
			LastName:  m.Sender.LastName,
		}
	}
	if option == current.Answer {
		score.Score += 1
	}
	self.storage.UpdateScore(m.Sender.ID, score)
	log.Printf("Score: %d", score.Score)
	currentQuestion.CurrentQuestion++
	self.storage.UpdateQuestion(m.Chat.ID, currentQuestion)
	self.next(m)
}

func (self Bot) checkRequirement(m *tb.Message) bool {
	chat, err := self.bot.ChatByID(CHAT_GROUP)
	if err != nil {
		log.Printf("Cannot get chat by id %s: %s", "testbot987", err.Error())
		return false
	}
	log.Printf("%+v", chat)
	qualified, err := self.bot.ChatMemberOf(chat, m.Sender)
	if err != nil {
		log.Printf("Cannot get chat member of: %s", err.Error())
		return false
	}
	if qualified.Role == tb.Creator || qualified.Role == tb.Administrator || qualified.Role == tb.Member {
		return true
	}
	return false
}

func (self Bot) handleStart(m *tb.Message) {
	// make sure user chat private to answer the question
	if !m.Private() {
		self.bot.Reply(m, "Bạn cần chat riêng với @KyberQuestionBot để trả lời câu hỏi vào tham gia bốc thăm may mắn :D")
		return
	}

	// make sure user joined require group to answer the question
	qualified := self.checkRequirement(m)
	if !qualified {
		self.bot.Send(m.Chat, "Bạn cần tham gia group @testbot987 để có thể tham gia chương trình.")
		return
	}

	message := "Bạn cần trả lời đúng cả 5 câu hỏi để được tham gia bốc thăm may mắn."
	self.bot.Send(m.Chat, message)
	// random a new sequence of question
	rand.Seed(time.Now().UnixNano())
	rands := rand.Perm(10)[:5]

	// remove question
	self.storage.RemoveQuestion(m.Chat.ID)

	// update new question
	currentQuestion, _ := self.storage.GetCurrentQuestion(m.Chat.ID)
	currentQuestion.ID = m.Chat.ID
	currentQuestion.Rands = rands
	currentQuestion.CurrentQuestion = 0
	self.storage.UpdateQuestion(m.Chat.ID, currentQuestion)

	// reset score
	self.storage.RemoveScore(m.Sender.ID)

	// start sending question
	self.next(m)
}

func (self Bot) handleWho(m *tb.Message) {
	payload := m.Payload
	if payload == "" {
		updateCurrentCommand("who", m)
		self.bot.Reply(m, "Bạn muốn check người may mắn cho số nào?")
	} else {
		self.handleCheckWho(m, payload)
	}
}
