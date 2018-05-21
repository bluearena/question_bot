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
	Key       string `json:"bot_key"`
	Deadline  int64  `json:"deadline"`
	ChatGroup string `json:"chatgroup"`
}

// Bot object
type Bot struct {
	bot      *tb.Bot
	storage  *QuestionStorage
	deadline int64
}

// Questions question list
type Questions []struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   int      `json:"answer"`
}

// const for chat group
var chatGroup string

// map[chatID]questionID
var questions Questions
var lucky map[string]string
var selectedNumber map[string]string
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
	chatGroup = botConfig.ChatGroup
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
		bot:      tbot,
		storage:  storage,
		deadline: botConfig.Deadline,
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
		mybot.handleUserJoined(m)
	})

	// mybot.bot.Handle(tb.OnUserLeft, func(m *tb.Message) {
	// 	mybot.handleUserLeft(m)
	// })

	mybot.bot.Handle("/top", func(m *tb.Message) {
		mybot.handleTop(m)
	})

	mybot.bot.Handle("/help", func(m *tb.Message) {
		mybot.handleHelp(m)
	})

	mybot.bot.Handle("/prize", func(m *tb.Message) {
		mybot.handlePrize(m)
	})

	mybot.bot.Handle("/yes", func(m *tb.Message) {
		mybot.handleYes(m)
	})

	mybot.bot.Handle("/no", func(m *tb.Message) {
		mybot.handleNo(m)
	})

	mybot.bot.Handle("/close", func(m *tb.Message) {
		mybot.handleClose(m)
	})

	mybot.bot.Start()
}

func (b Bot) handlePrize(m *tb.Message) {
	message := fmt.Sprintf(`Con thân mến, cơ cấu giải thưởng của chương trình như sau:

		⭐️️️ Ta có *15 giải* cho những người có vé số may mắn trong đó:

			💰 5 Giải đặc biệt mỗi giải 100 KNC
			💰 10 Giải mỗi giải 10 KNC
		
		⭐ Ngoài ra còn có *5 Giải* "cống hiến" mỗi giải là 40 KNC dành cho 5 thành viên mời được nhiều bạn tham gia nhất

	Chúc con may mắn 😉`)
	b.bot.Send(m.Chat, message, &tb.SendOptions{
		// ParseMode: tb.ModeMarkdown,
	})
}

func (b Bot) handleHelp(m *tb.Message) {
	message := fmt.Sprintf(`Chào con, Bụt đây.
	Con có thể /start để bắt đầu trả lời câu hỏi. Trả lời đúng hết cả 5 câu hỏi, Bụt sẽ thưởng cho con 1 "vé" để chọn số may mắn.
	Con có thể mời bạn bè vào @%s, để được tặng thêm "vé" may mắn, tăng khả năng trúng thưởng nhé.
	   
	/me để xem lại số vé may mắn con đã chọn,
	/top để xem xem ai mời nhiều nhất nè
	/who [số] để kiểm tra xem có ai chọn trùng số không.
	/prize để xem danh sách quà tặng của Bụt nhé.`, chatGroup)
	b.bot.Send(m.Chat, message)
}

func updateCurrentCommand(command string, m *tb.Message) {
	if len(lucky) == 0 {
		lucky = map[string]string{}
	}
	lucky[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)] = command
}

func updateSelectedNumber(lucky string, m *tb.Message) {
	if len(selectedNumber) == 0 {
		selectedNumber = map[string]string{}
	}
	selectedNumber[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)] = lucky
}

func (b Bot) activateUser(userID int) error {
	// activate score
	score, err := b.storage.GetUserScore(userID)
	if err == nil {
		score.Valid = true
		b.storage.UpdateScore(userID, score)
	}
	// activate invite member
	inviteUsers, err := b.storage.GetInvitedUser(userID)
	if err == nil {
		for _, user := range inviteUsers {
			user.Valid = true
			b.storage.UpdateInviteUser(user)
		}
	}
	// activate top score
	top, err := b.storage.GetTopByUserID(userID)
	if err == nil {
		top.Valid = true
		b.storage.UpdateTopObject(top)
	}
	return err
}

func (b Bot) deactivateUser(userID int) error {
	var err error
	// deactivate score
	score, err := b.storage.GetUserScore(userID)
	if err == nil {
		score.Valid = false
		b.storage.UpdateScore(userID, score)
	}
	// deactivate invite member
	inviteUsers, err := b.storage.GetInvitedUser(userID)
	if err == nil {
		for _, user := range inviteUsers {
			user.Valid = false
			b.storage.UpdateInviteUser(user)
		}
	}
	// deactivate top score
	top, err := b.storage.GetTopByUserID(userID)
	if err == nil {
		top.Valid = false
		b.storage.UpdateTopObject(top)
	}
	return err
}

func (b Bot) checkAlreadyInvited(user tb.User, m *tb.Message) {
	invitedUser, err := b.storage.GetInvitedUserByInvitedID(user.ID)
	if err == nil {
		if invitedUser.UserID != m.Sender.ID {
			b.storage.RemoveUser(user.ID)
			message := fmt.Sprintf("Bạn [%s](tg://user?id=%d đã rời khỏi group và được mời lại bởi 1 người khác, số may mắn con chọn cho bạn này không còn giá trị nữa.", invitedUser.InvitedName, invitedUser.InvitedID)
			user := &tb.User{
				ID: invitedUser.UserID,
			}
			b.bot.Send(user, message)
		}
	}
}

func (b Bot) handleUserJoined(m *tb.Message) {
	if time.Now().Unix() > b.deadline {
		return
	}
	if m.Chat.Username != chatGroup {
		return
	}
	if m.Sender.ID == m.UserJoined.ID {
		b.activateUser(m.UserJoined.ID)
		return
	}
	message := "Con đã add "
	for _, user := range m.UsersJoined {
		b.checkAlreadyInvited(user, m)
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
			Valid:           true,
		}
		b.storage.InvitedUser(m.Sender.ID, inviteUser)
		b.storage.UpdateTop(m.Sender.ID, name, 1)
	}
	message += fmt.Sprintf(" vào group @%s. Con được thêm %d lần chọn số may mắn. Con có thể /add để thêm số may mắn nhé.", chatGroup, len(m.UsersJoined))
	b.bot.Send(m.Sender, message, &tb.SendOptions{
		ParseMode: tb.ModeMarkdown,
	})

	// update valid if this user used to be in the group (and join the campaign)
	for _, user := range m.UsersJoined {
		b.activateUser(user.ID)
	}
}

func (b Bot) handleMe(m *tb.Message) {
	score, _ := b.storage.GetUserScore(m.Sender.ID)
	if score.ID == 0 {
		b.bot.Reply(m, "Con chưa tham gia trả lời câu hỏi. Hãy chat /start riêng với Bụt để tham gia trả lời câu hỏi và có cơ hội nhận quà nhé.")
		return
	}
	if !m.Private() {
		b.bot.Reply(m, "Bụt sẽ trả lời riêng cho con.")
	}
	message := ""
	invites, err := b.storage.GetInvitedUser(m.Sender.ID)
	log.Printf("Score: %+v", score)
	if (score.ID != 0 && score.Valid == false) || (err == nil && invites[0].Valid == false) {
		message += fmt.Sprintf("Rất tiếc con đã rời khỏi group @%s. Kết quả dưới đây của con không được tính. \n", chatGroup)
	}
	if score.Score == 5 {
		message += fmt.Sprintf("Con đã trả lời chính xác %d/5 câu hỏi và số may mắn con đã chọn là: %s\n", score.Score, score.LuckyNumber)
	} else {
		message += fmt.Sprintf("Con đã trả lời chính xác %d/5 câu hỏi, con chưa được chọn số may mắn.\n", score.Score)
	}
	if err != nil && err.Error() == "not found" {
		message += fmt.Sprintf("Con hãy mời thêm người bạn nào vào @%s để nhận được thêm vé may mắn nhé 🤗. \n", chatGroup)
	} else {
		message += fmt.Sprintf("Con đã mời: \n")
		for _, user := range invites {
			name := strings.TrimSpace(user.InvitedName)
			message += fmt.Sprintf("[%s](tg://user?id=%d), số may mắn: %s \n", name, user.InvitedID, user.LuckyNumber)
		}
	}
	_, err = b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		message += fmt.Sprintf("Con có thể /add để thêm số may mắn.")
	}
	b.bot.Send(m.Sender, message, &tb.SendOptions{
		ParseMode: tb.ModeMarkdown,
	})
}

func (b Bot) handleAdd(m *tb.Message) {
	if time.Now().Unix() > b.deadline {
		b.bot.Reply(m, "Bụt rất tiếc, thời gian tham gia chương trình đã hết.")
		return
	}
	if !m.Private() {
		b.bot.Reply(m, "/add riêng cho Bụt để Bụt thêm số may mắn cho.")
		return
	}
	score, _ := b.storage.GetUserScore(m.Sender.ID)
	if score.Score == 5 && score.LuckyNumber == "" {
		updateCurrentCommand("invited", m)
		b.bot.Send(m.Sender, "Điền 4 chữ số may mắn: ")
		return
	}
	_, err := b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err == nil {
		updateCurrentCommand("invited", m)
		b.bot.Send(m.Sender, "Điền 4 chữ số may mắn: ")
	} else {
		b.bot.Send(m.Sender, "Con không còn vé nào để chọn số may mắn.")
	}
}

func (b Bot) handleTop(m *tb.Message) {
	users, err := b.storage.GetTop()
	if err == nil {
		message := "Top 5 người mời nhiều bạn bè nhất: \n"
		count := 0
		for i := len(users); i > 0; i-- {
			log.Printf("Invites: %+v", users[i-1])
			if users[i-1].Valid == false {
				continue
			}
			if count++; count > 5 {
				break
			}
			message += fmt.Sprintf("[%s](tg://user?id=%d) - %d người\n", users[i-1].Name, users[i-1].ID, users[i-1].Point)
		}
		if count == 0 {
			message += "Chưa có ai trong danh sách top"
		}
		b.bot.Send(m.Chat, message, &tb.SendOptions{
			ParseMode: tb.ModeMarkdown,
		})
	} else {
		b.bot.Send(m.Chat, "Chưa có ai trong danh sách top")
	}
}

func (b Bot) handleUserLeft(m *tb.Message) {
	if m.Chat.Username != chatGroup {
		return
	}
	b.deactivateUser(m.UserLeft.ID)
	receiver := tb.User{}
	message := ""
	if m.UserLeft.ID == m.Sender.ID {
		receiver = tb.User{
			ID: m.UserLeft.ID,
		}
		message = fmt.Sprintf("Sao con lại rời khỏi group @%s. Buồn quá, Bụt phải cho con ra khỏi danh sách nhận quà rồi 😢", chatGroup)
	} else {
		exist, err := b.storage.GetInvitedUserByInvitedID(m.UserLeft.ID)
		if err == nil {
			b.storage.RemoveUser(m.UserLeft.ID)
			message = fmt.Sprintf("[%s](tg://user?id=%d) đã rời khỏi group @%s. Số may mắn con chọn cho [%s](tg://user?id=%d) đã không còn hiệu lực nữa.",
				exist.InvitedName, exist.InvitedID, chatGroup, exist.InvitedName, exist.InvitedID)
			receiver = tb.User{
				ID: exist.UserID,
			}
			b.storage.UpdateTop(exist.UserID, exist.Username, -1)
		}
	}
	b.bot.Send(&receiver, message, &tb.SendOptions{
		ParseMode: tb.ModeMarkdown,
	})
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
		b.handleUpdateLucky(m)
	case "who":
		b.handleCheckWho(m, m.Text)
	case "invited":
		b.handleInvited(m)
	default:
		b.handleDefault(m)
	}
}

func (b Bot) handleDefault(m *tb.Message) {
	if m.Private() {
		b.bot.Send(m.Chat, `Con nói gì Bụt không hiểu. Bấm /help để nhận được hướng dẫn nhé.`)
	}
}

func (b Bot) checkDuplicate(userID int, lucky string) bool {
	score, _ := b.storage.GetUserScore(userID)
	if score.LuckyNumber == lucky {
		return true
	}
	invitedUsers, _ := b.storage.GetInvitedUser(userID)
	for _, user := range invitedUsers {
		if user.LuckyNumber == lucky {
			return true
		}
	}
	return false
}

func (b Bot) handleDuplicate(m *tb.Message, lucky string) {
	message := "Con đã chọn số này, con có chắc vẫn muốn chọn số này lần nữa? /yes để tiếp tục chọn /no để chọn lại số khác."
	updateSelectedNumber(lucky, m)
	b.bot.Reply(m, message)
}

func (b Bot) handleYes(m *tb.Message) {
	if !m.Private() {
		return
	}
	if len(selectedNumber) == 0 {
		return
	}
	lucky := selectedNumber[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)]
	if lucky == "" {
		return
	}
	invitedUser, err := b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
	if err != nil {
		log.Printf("Cannot get invited: %s", err.Error())
	}
	invitedUser[0].LuckyNumber = lucky
	err = b.storage.UpdateInviteUser(invitedUser[0])
	if err != nil {
		log.Printf("Cannot update lucky number: %s", err.Error())
	}
	message := fmt.Sprintf("Số may mắn con đã chọn là: %s, Bụt sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc nhé. ", lucky)
	if len(invitedUser) > 1 {
		message += fmt.Sprintf("Con còn %d vé, /add để chọn số may mắn nhé.", len(invitedUser)-1)
	}
	b.bot.Send(m.Chat, message)
}

func (b Bot) handleNo(m *tb.Message) {
	lucky := selectedNumber[fmt.Sprintf("%d_%d", m.Chat.ID, m.Sender.ID)]
	if lucky == "" {
		return
	}
	updateSelectedNumber("", m)
	updateCurrentCommand("invited", m)
	message := "Số con chọn đã bị hủy, hãy chọn số may mắn mới."
	b.bot.Reply(m, message)
}

func (b Bot) handleInvited(m *tb.Message) {
	text := strings.TrimSpace(m.Text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Con phải gửi 4 chữ số thì Bụt mới lưu lại được."))
	} else {
		if b.checkDuplicate(m.Sender.ID, text) {
			b.handleDuplicate(m, text)
			updateCurrentCommand("", m)
			return
		}
		score, _ := b.storage.GetUserScore(m.Sender.ID)
		invitedUser, err := b.storage.GetInvitedUserWithoutLuckyNumber(m.Sender.ID)
		if err != nil {
			log.Printf("Cannot get invited: %s", err.Error())
		}
		if score.Score == 5 && score.LuckyNumber == "" {
			score.LuckyNumber = text
			b.storage.UpdateScore(m.Sender.ID, score)
		} else {
			invitedUser[0].LuckyNumber = text
			err = b.storage.UpdateInviteUser(invitedUser[0])
			if err != nil {
				log.Printf("Cannot update lucky number: %s", err.Error())
			}
		}
		message := fmt.Sprintf("Số may mắn con đã chọn là: %s, Bụt sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc nhé. ", text)
		if len(invitedUser) > 1 {
			message += fmt.Sprintf("Con còn %d vé, /add để chọn số may mắn nhé.", len(invitedUser)-1)
		}
		b.bot.Send(m.Chat, message)
		updateCurrentCommand("", m)
	}
}

func (b Bot) handleUpdateLucky(m *tb.Message) {
	if time.Now().Unix() > b.deadline {
		b.bot.Reply(m, "Bụt rất tiếc, thời gian tham gia chương trình đã hết.")
		return
	}
	text := strings.TrimSpace(m.Text)
	matched, err := regexp.MatchString(`^\d{4,4}$`, text)
	if err != nil {
		log.Printf("Cannot match: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Con phải gửi 4 chữ số thì Bụt mới lưu lại được."))
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
		message := fmt.Sprintf("Số may mắn con đã chọn là: %s, bụt sẽ quay số may mắn và thông báo người trúng thưởng khi chương trình kết thúc.", score.LuckyNumber)
		message += fmt.Sprintf("Con hãy mời thêm bạn nào vào @%s để nhận được thêm vé may mắn nhé 🤗.", chatGroup)
		b.bot.Send(m.Chat, message)
		updateCurrentCommand("", m)
	}
}

func (b Bot) handleCheckWho(m *tb.Message, luckyNumber string) {
	luckyStr := strings.TrimSpace(luckyNumber)
	matched, err := regexp.MatchString(`^\d{4,4}$`, luckyStr)
	if err != nil {
		log.Printf("Cannot match lucky string: %s", err.Error())
	}
	if !matched {
		b.bot.Reply(m, fmt.Sprintf("Con phải gửi 4 chữ số thì Bụt mới tìm được."))
	} else {
		updateCurrentCommand("", m)
		users, err := b.storage.Who(luckyStr)
		if err != nil && err.Error() != "not found" {
			log.Printf("Cannot get user: %s", err)
		}
		// if err != nil && err.Error() == "not found" {
		// 	b.bot.Reply(m, fmt.Sprintf("Chưa có người dùng nào chọn số %s.", luckyStr))
		// 	return
		// }
		message := ""
		if len(users) != 0 {
			if users[0].LuckyNumber == luckyStr {
				message = fmt.Sprintf("Danh sách những người đã chọn số %s: \n\n", luckyStr)
			} else {
				message = fmt.Sprintf("Chưa có ai chọn số %s, người chọn gần nhất là: \n\n", luckyStr)
			}
		} else {
			message = fmt.Sprintf("Chưa có ai trong danh sách.")
		}
		for _, user := range users {
			message += fmt.Sprintf("[%s](tg://user?id=%d) - số đã chọn: %s \n", user.Name, user.ID, user.LuckyNumber)
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
		b.bot.Send(m.Sender, message, &tb.SendOptions{
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
	message := fmt.Sprintf("Con đã trả lời đúng: %d/5 câu hỏi.\n", score.Score)
	if score.Score == 5 {
		message += fmt.Sprintf("Thông minh quá. Nhập 4 chữ số để Bụt quay số may mắn nào.")
		updateCurrentCommand("lucky", m)
	} else {
		message += fmt.Sprintf("Tiếc quá cơ, con chưa trả lời được cả 5 câu hỏi. Thử lại để đạt mức điểm cao hơn: /start")
	}

	b.bot.Send(m.Chat, message,
		&tb.SendOptions{
			ReplyMarkup: &tb.ReplyMarkup{
				ReplyKeyboardRemove: true,
			},
		})
}

func (b Bot) handleAnswer(m *tb.Message, option int) {
	currentQuestion, _ := b.storage.GetCurrentQuestion(m.Chat.ID)
	current := questions[currentQuestion.Rands[currentQuestion.CurrentQuestion]]
	if option+1 > len(current.Options) {
		b.bot.Send(m.Chat, fmt.Sprintf("Câu hỏi không có phương án con chọn."))
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
			Valid:     true,
		}
	}
	if option == current.Answer {
		score.Score++
	}
	b.storage.UpdateScore(m.Sender.ID, score)
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
	if time.Now().Unix() > b.deadline {
		b.bot.Reply(m, "Bụt rất tiếc, thời gian tham gia chương trình đã hết.")
		return
	}
	// make sure user chat private to answer the question
	if !m.Private() {
		b.bot.Reply(m, "Con cần chat riêng với @KyberQuestionBot để trả lời câu hỏi vào tham gia bốc thăm may mắn :D")
		return
	}

	// make sure user joined require group to answer the question
	qualified := b.checkRequirement(m)
	if !qualified {
		b.bot.Send(m.Chat, fmt.Sprintf("Con cần tham gia group @%s để có thể tham gia chương trình.", chatGroup))
		return
	}

	message := "Con chỉ cần trả lời đúng 5 câu hỏi đơn giản của Bụt để được tham gia bốc thăm may mắn."
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
			b.bot.Reply(m, "Con muốn kiểm người may mắn cho số nào?")
		} else {
			b.bot.Reply(m, "Sử dụng cú pháp /who [số] để kiểm tra số may mắn trong group nhé")
		}
	} else {
		b.handleCheckWho(m, payload)
	}
}

func (b Bot) handleClose(m *tb.Message) {
	chat, err := b.bot.ChatByID("@" + chatGroup)
	if err != nil {
		log.Printf("Cannot get chat by id %s: %s", chatGroup, err.Error())
		return
	}
	qualified, err := b.bot.ChatMemberOf(chat, m.Sender)
	if err != nil {
		log.Printf("Cannot get chat member of: %s", err.Error())
		return
	}
	if qualified.Role == tb.Creator || qualified.Role == tb.Administrator {
		// validate user score
		scores, err := b.storage.GetAllUserScore()
		if err == nil {
			for _, score := range scores {
				u := &tb.User{
					ID: score.ID,
				}
				member, err := b.bot.ChatMemberOf(chat, u)
				if err != nil {
					continue
				}
				if member.Role == tb.Creator || member.Role == tb.Administrator || member.Role == tb.Member {
					continue
				}
				score.Valid = false
				b.storage.UpdateScore(score.ID, score)
				top, err := b.storage.GetTopByUserID(score.ID)
				if err == nil {
					top.Valid = false
					b.storage.UpdateTopObject(top)
				}
			}
		}
		// validate user invites
		inviteUsers, err := b.storage.GetAllInvitedUser()
		if err == nil {
			for _, user := range inviteUsers {
				u := &tb.User{
					ID: user.InvitedID,
				}
				member, err := b.bot.ChatMemberOf(chat, u)
				if err != nil {
					continue
				}
				if member.Role == tb.Creator || member.Role == tb.Administrator || member.Role == tb.Member {
					continue
				}
				b.storage.RemoveUser(user.InvitedID)
				b.storage.UpdateTop(user.UserID, user.Name, -1)
			}
		}
	}
}
