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

// BotConfig config for bot
type BotConfig struct {
	Key string `json:"bot_key"`
}

// Bot object
type Bot struct {
	bot     *tb.Bot
	storage *QuestionStorage
}

// Questions question list
type Questions []struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   int      `json:"answer"`
}

// const for chat group
const chatGroup string = "testquestion1234"

// map[chatID]questionID
var questions Questions
var lucky map[string]string
var replyKeysTwo [][]tb.ReplyButton
var replyKeysFour [][]tb.ReplyButton

func readConfigFromFile(path string) (BotConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return BotConfig{}, err
	}
	result := BotConfig{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func readQuestionsFromFile(path string) (Questions, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return Questions{}, err
	}
	result := Questions{}
	err = json.Unmarshal(data, &result)
	return result, err

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

	mybot.bot.Handle("/top", func(m *tb.Message) {
		mybot.handleTop(m)
	})

	mybot.bot.Handle("/help", func(m *tb.Message) {
		mybot.handleHelp(m)
	})

	mybot.bot.Start()
}

func (b Bot) handleHelp(m *tb.Message) {
	message := fmt.Sprintf(`Chào con, Bụt đây.
	Con có thể /start để bắt đầu trả lời câu hỏi. Trả lời đúng hết cả 5 câu hỏi của Bụt để được chọn số may mắn.
	Mời bạn bè vào @%s, để được chọn thêm số may mắn, tăng khả năng trúng thưởng.
	5 người mời nhiều người nhất sẽ có quà nhé.
	/me để xem bản thân được bao nhiêu điểm này,
	/top để xem xem ai mời nhiều nhất nè
	/who [số] để kiểm tra xem có ai chọn trùng số không.`, chatGroup)
	b.bot.Send(m.Chat, message)
}

func updateCurrentCommand(command string, m *tb.Message) {
	if len(lucky) == 0 {
		lucky = map[string]string{}
	}
	lucky[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)] = command
}

func (b Bot) handleUserJoined(m *tb.Message) {
	if m.Sender.ID == m.UserJoined.ID || m.Chat.Username != chatGroup {
		return
	}
	message := "Bạn đã add "
	for _, user := range m.UsersJoined {
		name := fmt.Sprintf("%s %s", m.Sender.FirstName, m.Sender.LastName)
		invitedName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		message += fmt.Sprintf("[%s](tg://user?id=%d)", invitedName, m.UserJoined.ID)
		inviteUser := InviteUser{
			UserID:          m.Sender.ID,
			InvitedID:       user.ID,
			LuckyNumber:     "",
			Username:        m.Sender.Username,
			InvitedUsername: user.Username,
			Name:            name,
			InvitedName:     invitedName,
		}
		b.storage.InvitedUser(m.Sender.ID, inviteUser)
		b.storage.UpdateTop(m.Sender.ID, name, 1)
	}
	message += fmt.Sprintf(" vào group @%s. Bạn được thêm %d lần chọn số may mắn. Bạn có thể /add để thêm số may mắn.", chatGroup, len(m.UsersJoined))
	b.bot.Send(m.Sender, message, &tb.SendOptions{
		ParseMode: tb.ModeMarkdown,
	})
}

func (b Bot) handleMe(m *tb.Message) {
	if !m.Private() {
		b.bot.Reply(m, "Chúng tôi sẽ trả lời riêng cho bạn.")
	}
	score, _ := b.storage.GetUserScore(m.Sender.ID)
	message := fmt.Sprintf("Trả lời câu hỏi: %d/5, số may mắn: %s\n", score.Score, score.LuckyNumber)
	invites, err := b.storage.GetInvitedUser(m.Sender.ID)
	if err != nil && err.Error() == "not found" {
		message += fmt.Sprintf("Bạn chưa mời thêm người bạn nào vào @%s. \n", chatGroup)
	} else {
		message += fmt.Sprintf("Bạn đã mời: \n")
		for _, user := range invites {
			message += fmt.Sprintf("[%s](tg://user?id=%d), số may mắn: %s \n", user.InvitedName, user.InvitedID, user.LuckyNumber)
		}
	}
	_, err = b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		message += fmt.Sprintf("Bạn có thể /add để thêm số may mắn.")
	}
	b.bot.Send(m.Sender, message, &tb.SendOptions{
		ParseMode: tb.ModeMarkdown,
	})
}

func (b Bot) handleAdd(m *tb.Message) {
	if !m.Private() {
		b.bot.Reply(m, "Chúng tôi sẽ trả lời riêng cho bạn.")
	}
	_, err := b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		updateCurrentCommand("invited", m)
		b.bot.Send(m.Sender, "Điền 4 chữ số may mắn: ")
	} else {
		b.bot.Send(m.Sender, "Bạn không còn lượt chọn số may mắn nào.")
	}
}

func (b Bot) handleTop(m *tb.Message) {
	users, err := b.storage.GetTop()
	if err == nil {
		log.Printf("Top: %+v", users)
		message := "Top 5 người invite nhiều nhất: \n"
		count := 0
		for i := len(users); i > 0; i-- {
			if count++; count > 5 {
				break
			}
			message += fmt.Sprintf("[%s](tg://user?id=%d) - điểm: %d\n", users[i-1].Name, users[i-1].ID, users[i-1].Point)
		}
		b.bot.Send(m.Chat, message, &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	} else {
		log.Printf("Top error: %s", err.Error())
		b.bot.Send(m.Chat, "Chưa có ai trong danh sách top")
	}
}

func (b Bot) handleUserLeft(m *tb.Message) {
	// TODO: if user left remove all their point
	if m.Chat.Username != chatGroup {
		return
	}
	exist, err := b.storage.GetInvitedUserByInvitedID(m.UserLeft.ID)
	if err == nil {
		b.storage.RemoveUser(m.UserLeft.ID)
		message := fmt.Sprintf("[%s](tg://user?id=%d) đã rời khỏi group @%s. Số may mắn bạn chọn cho [%s](tg://user?id=%d) đã bị hủy.", exist.InvitedName, exist.InvitedID, chatGroup, exist.InvitedName, exist.InvitedID)
		user := tb.User{
			ID: exist.UserID,
		}
		b.storage.UpdateTop(exist.UserID, exist.Username, -1)
		b.bot.Send(&user, message, &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	}
}

func (b Bot) initReplyKeys(questionOptions []string) [][]tb.ReplyButton {
	replyKeys := [][]tb.ReplyButton{}
	replyKeyOne := []tb.ReplyButton{}
	for key := range questionOptions {
		replyBtn := tb.ReplyButton{Text: questionOptions[key]}
		b.bot.Handle(&replyBtn, func(m *tb.Message) {
			option := 0
			for i, v := range questionOptions {
				if v == replyBtn.Text {
					option = i
				}
			}
			b.handleAnswer(m, option)
		})
		replyKeyOne = append(replyKeyOne, replyBtn)
		if key%2 == 1 {
			replyKeys = append(replyKeys, replyKeyOne)
			replyKeyOne = []tb.ReplyButton{}
		}
	}
	return replyKeys
}

func (b Bot) handleText(m *tb.Message) {
	switch lucky[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)] {
	case "lucky":
		b.handleUpateLucky(m)
	case "who":
		b.handleCheckWho(m, m.Text)
	case "invited":
		b.handleInvited(m)
	}
}

func (b Bot) handleInvited(m *tb.Message) {
	text := strings.TrimSpace(m.Text)
	log.Printf("Invited: %s", text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số."))
	} else {
		invitedUser, err := b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
		if err != nil {
			log.Printf("Cannot get score: %s", err.Error())
		}
		invitedUser.LuckyNumber = text
		err = b.storage.UpdateInviteUser(invitedUser)
		if err != nil {
			log.Printf("Cannot update lucky number: %s", err.Error())
		}
		b.bot.Send(m.Chat, fmt.Sprintf("Con số may mắn bạn đã chọn là: %s, chúng tôi sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc.", text))
		updateCurrentCommand("", m)
	}
}

func (b Bot) handleUpateLucky(m *tb.Message) {
	text := strings.TrimSpace(m.Text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số."))
	} else {
		score, err := b.storage.GetUserScore(m.Sender.ID)
		if err != nil {
			log.Printf("Cannot get score: %s", err.Error())
		}
		score.LuckyNumber = text
		err = b.storage.UpdateScore(m.Sender.ID, score)
		if err != nil {
			log.Printf("Cannot update lucky number: %s", err.Error())
		}
		score, _ = b.storage.GetUserScore(m.Sender.ID)
		b.bot.Send(m.Chat, fmt.Sprintf("Con số may mắn bạn đã chọn là: %s, chúng tôi sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc.", score.LuckyNumber))
		updateCurrentCommand("", m)
	}
}

func (b Bot) handleCheckWho(m *tb.Message, luckyNumber string) {
	luckyStr := strings.TrimSpace(luckyNumber)
	matched, err := regexp.MatchString(`^\d{4,4}$`, luckyStr)
	log.Printf("%+v", matched)
	if err != nil {
		log.Printf("Cannot match lucky string: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Bạn phải gửi 4 chữ số để kiểm tra người may mắn."))
	} else {
		updateCurrentCommand("", m)
		users, err := b.storage.Who(luckyStr)
		if err != nil && err.Error() != "not found" {
			log.Printf("Cannot get user: %s", err)
		}
		if err != nil && err.Error() == "not found" {
			b.bot.Reply(m, fmt.Sprintf("Chưa có người dùng nào chọn số %s.", luckyStr))
			return
		}
		message := fmt.Sprintf("Danh sách những người đã chọn số %s: \n\n", luckyStr)
		for _, user := range users {
			message += fmt.Sprintf("[%s](tg://user?id=%d) \n", user.Name, user.ID)
		}
		b.bot.Reply(m, message, &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	}
}

// send the next question
func (b Bot) next(m *tb.Message) {
	currentQuestion, _ := b.storage.GetCurrentQuestion(m.Chat.ID)
	nextQuestion := currentQuestion.CurrentQuestion
	rands := currentQuestion.Rands

	if nextQuestion+1 > len(rands) {
		b.finish(m)
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
		b.bot.Send(m.Chat, message, &tb.SendOptions{
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

func (b Bot) finish(m *tb.Message) {
	score, _ := b.storage.GetUserScore(m.Sender.ID)
	message := fmt.Sprintf("Bạn đã hoàn thành. Bạn trả lời đúng: %d/5 câu hỏi.\n", score.Score)
	if score.Score == 5 {
		message += fmt.Sprintf("Nhập 4 chữ số để quay số may mắn.")
		updateCurrentCommand("lucky", m)
	} else {
		message += fmt.Sprintf("Rất tiếc bạn chưa đủ điều kiện quay số may mắn. Thử lại để đạt mức điểm cao hơn: /start")
	}

	b.bot.Send(m.Chat, message,
		&tb.SendOptions{
			ReplyMarkup: &tb.ReplyMarkup{
				ReplyKeyboardRemove: true,
			},
		})
}

func (b Bot) handleAnswer(m *tb.Message, option int) {
	log.Printf("%d", option)
	currentQuestion, _ := b.storage.GetCurrentQuestion(m.Chat.ID)
	log.Printf("%+v", currentQuestion)
	current := questions[currentQuestion.Rands[currentQuestion.CurrentQuestion]]
	log.Printf("%+v", current)
	if option+1 > len(current.Options) {
		b.bot.Send(m.Chat, fmt.Sprintf("Câu hỏi không có phương án bạn chọn."))
		return
	}

	score, err := b.storage.GetUserScore(m.Sender.ID)
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
		score.Score++
	}
	b.storage.UpdateScore(m.Sender.ID, score)
	log.Printf("Score: %d", score.Score)
	currentQuestion.CurrentQuestion++
	b.storage.UpdateQuestion(m.Chat.ID, currentQuestion)
	b.next(m)
}

func (b Bot) checkRequirement(m *tb.Message) bool {
	chat, err := b.bot.ChatByID("@" + chatGroup)
	if err != nil {
		log.Printf("Cannot get chat by id %s: %s", chatGroup, err.Error())
		return false
	}
	log.Printf("%+v", chat)
	qualified, err := b.bot.ChatMemberOf(chat, m.Sender)
	if err != nil {
		log.Printf("Cannot get chat member of: %s", err.Error())
		return false
	}
	if qualified.Role == tb.Creator || qualified.Role == tb.Administrator || qualified.Role == tb.Member {
		return true
	}
	return false
}

func (b Bot) handleStart(m *tb.Message) {
	// make sure user chat private to answer the question
	if !m.Private() {
		b.bot.Reply(m, "Bạn cần chat riêng với @KyberQuestionBot để trả lời câu hỏi vào tham gia bốc thăm may mắn :D")
		return
	}

	// make sure user joined require group to answer the question
	qualified := b.checkRequirement(m)
	if !qualified {
		b.bot.Send(m.Chat, fmt.Sprintf("Bạn cần tham gia group @%s để có thể tham gia chương trình.", chatGroup))
		return
	}

	message := "Bạn cần trả lời đúng cả 5 câu hỏi để được tham gia bốc thăm may mắn."
	b.bot.Send(m.Chat, message)
	// random a new sequence of question
	rand.Seed(time.Now().UnixNano())
	rands := rand.Perm(10)[:5]

	// remove question
	b.storage.RemoveQuestion(m.Chat.ID)

	// update new question
	currentQuestion, _ := b.storage.GetCurrentQuestion(m.Chat.ID)
	currentQuestion.ID = m.Chat.ID
	currentQuestion.Rands = rands
	currentQuestion.CurrentQuestion = 0
	b.storage.UpdateQuestion(m.Chat.ID, currentQuestion)

	// reset score
	b.storage.RemoveScore(m.Sender.ID)

	// start sending question
	b.next(m)
}

func (b Bot) handleWho(m *tb.Message) {
	payload := m.Payload
	if payload == "" {
		if m.Private() {
			updateCurrentCommand("who", m)
			b.bot.Reply(m, "Bạn muốn check người may mắn cho số nào?")
		} else {
			b.bot.Reply(m, "Sử dụng cú pháp /who [số] để kiểm tra số may mắn trong group nhé")
		}
	} else {
		b.handleCheckWho(m, payload)
	}
}
