package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/arkhipovkm/unifeed-go/db"
	"github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/zelenin/go-tdlib/client"
)

type Replics struct {
	Help                           string
	SubscriptionsHeader            string
	SubscriptionsFooter            string
	SubscriptionsDefault           string
	SuccessfullySubscribed         string
	AlreadySubscribed              string
	ReplyToSubscribed              string
	ReplyToNoLongerSubscribed      string
	ReplyButtonSubscribe           string
	ReplyButtonUnsubscribe         string
	AnswerSuccessfullySubscribed   string
	AnswerSuccessfullyUnsubscribed string
}

var replicsLangMap = map[string]*Replics{
	"ru": {
		Help:                           "Привет, это %s! Этот бот помогает вам собрать ваши любимые каналы в одну ленту. Вы можете подписаться на любой канал, просто переслав из него сообщение боту.\n\nКоманды:\n/subscriptions - список моих подписок",
		SubscriptionsHeader:            "Вы подписаны на следующие каналы:\n\n",
		SubscriptionsFooter:            "\nЧтобы отписаться, просто ответьте на любое сообщение из канала в ленте и нажмите \"Отписаться\".",
		SubscriptionsDefault:           "Вы пока ни на что не подписаны. Вы можете подписаться на любой канал, просто переслав из него сообщение боту.",
		SuccessfullySubscribed:         "Я добавил канал *%s* в список ваших подписок. Чтобы отписаться, ответьте на любое сообщение, пересланные с этого канала и нажмите \"Отписаться\".",
		AlreadySubscribed:              "Вы уже подписаны на канал *%s*. Чтобы отписаться, ответьте на любое сообщение, пересланные с этого канала и нажмите \"Отписаться\".",
		ReplyToSubscribed:              "Вы подписаны на %s",
		ReplyToNoLongerSubscribed:      "Вы больше не подписаны на %s",
		AnswerSuccessfullyUnsubscribed: "Вы успешно отписались от %s",
		AnswerSuccessfullySubscribed:   "Вы снова подписаны на %s",
		ReplyButtonSubscribe:           "Подписаться",
		ReplyButtonUnsubscribe:         "Отписаться",
	},
	"en": {
		Help:                           "Hi, its the %s! This bot helps you gather your favorite channels into one feed. To subscribe to a channel, simply forward me a message from there.\n\nCommands:\n/subscriptions - list the channels I'm subscribed to",
		SubscriptionsHeader:            "You're subscribed to the following channels:\n\n",
		SubscriptionsFooter:            "\nReply to a forwarded message to unsubscribe from that channel.",
		SubscriptionsDefault:           "You're subscribed to no channel yet. Forward me a message from any channel and I'll feed you here new posts from there!",
		SuccessfullySubscribed:         "I've added *%s* channel to your feed list. Reply to any message from this channel to unsubscribe",
		AlreadySubscribed:              "You've already subscribed to *%s*. \nReply to %s's messages to unsubscribe.\nEnlist /subscriptions.",
		ReplyToSubscribed:              "You're subscribed to %s",
		ReplyToNoLongerSubscribed:      "You're no longer subscribed to %s",
		AnswerSuccessfullyUnsubscribed: "Successfully unsubscribed from %s",
		AnswerSuccessfullySubscribed:   "Successfully re-subscribed to %s",
		ReplyButtonSubscribe:           "Subscribe",
		ReplyButtonUnsubscribe:         "Unsubscribe",
	},
}

func getReplics(update tgbotapi.Update) *Replics {
	var languageCode string
	if update.Message != nil && update.Message.From != nil {
		languageCode = update.Message.From.LanguageCode
	} else if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		languageCode = update.CallbackQuery.From.LanguageCode
	}

	var replics *Replics
	for k, v := range replicsLangMap {
		if k == languageCode {
			replics = v
		}
	}
	if replics == nil {
		replics = replicsLangMap["en"]
	}
	return replics
}

func processBot(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel, ch chan string) {
	var err error
	unifeedBotToUserChatID, _ := strconv.Atoi(os.Getenv("UNIFEED_BOT_TO_USER_CHAT_ID"))
	for update := range updates {
		replics := getReplics(update)
		if update.CallbackQuery != nil && update.CallbackQuery.Data != "" && update.CallbackQuery.Message != nil {
			var chatID int64
			if update.CallbackQuery.Message != nil &&
				update.CallbackQuery.Message.Chat != nil &&
				update.CallbackQuery.Message.Chat.ID != 0 {
				chatID = update.CallbackQuery.Message.Chat.ID
			} else if update.CallbackQuery.From != nil {
				chatID = int64(update.CallbackQuery.From.ID)
			} else {
				continue
			}

			var re *regexp.Regexp

			re = regexp.MustCompile("^unsubscribe-(.*?)$")
			if re.MatchString(update.CallbackQuery.Data) {
				parts := re.FindStringSubmatch(update.CallbackQuery.Data)
				if len(parts) == 0 {
					log.Println("Unsubscribe callback query does not contain channelUsername. Passing..")
					continue
				}
				channelUsername := parts[1]

				re = regexp.MustCompile("^Forwarded from (.*?)\\n\\n")
				parts = re.FindStringSubmatch(update.CallbackQuery.Message.ReplyToMessage.Text)
				var channelTitle string
				if len(parts) > 0 {
					channelTitle = parts[1]
				} else {
					log.Println("Could not resolve channelTitle from the Message.Text. Taking channelUsername as channelTitle..")
					channelTitle = channelUsername
				}

				err = db.DeleteChatChannel(int(chatID), channelUsername)
				if err != nil {
					log.Println(err)
				}

				callbackData := fmt.Sprintf("subscribe-%s", channelUsername)
				editMsg := &tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    chatID,
						MessageID: update.CallbackQuery.Message.MessageID,
						ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
							InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
								tgbotapi.InlineKeyboardButton{
									Text:         replics.ReplyButtonSubscribe,
									CallbackData: &callbackData,
								},
							}},
						},
					},
					Text: fmt.Sprintf(replics.ReplyToNoLongerSubscribed, channelTitle),
				}
				bot.Send(editMsg)
				bot.AnswerCallbackQuery(tgbotapi.NewCallback(
					update.CallbackQuery.ID,
					fmt.Sprintf(replics.AnswerSuccessfullyUnsubscribed, channelTitle),
				))
				continue
			}

			re = regexp.MustCompile("^subscribe-(.*?)$")
			if re.MatchString(update.CallbackQuery.Data) {
				parts := re.FindStringSubmatch(update.CallbackQuery.Data)
				if len(parts) == 0 {
					log.Println("Subscribe callback query does not contain channelUsername. Passing..")
					continue
				}
				channelUsername := parts[1]

				re = regexp.MustCompile("^Forwarded from (.*?)\\n\\n")
				parts = re.FindStringSubmatch(update.CallbackQuery.Message.ReplyToMessage.Text)
				var channelTitle string
				if len(parts) > 0 {
					channelTitle = parts[1]
				} else {
					channelTitle = channelUsername
				}

				err = db.PutChatChannel(int(chatID), channelUsername)
				if err != nil {
					log.Println(err)
				}

				callbackData := fmt.Sprintf("unsubscribe-%s", channelUsername)
				editMsg := &tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    chatID,
						MessageID: update.CallbackQuery.Message.MessageID,
						ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
							InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
								tgbotapi.InlineKeyboardButton{
									Text:         replics.ReplyButtonUnsubscribe,
									CallbackData: &callbackData,
								},
							}},
						},
					},
					Text: fmt.Sprintf(replics.ReplyToSubscribed, channelTitle),
				}
				bot.Send(editMsg)
				bot.AnswerCallbackQuery(tgbotapi.NewCallback(
					update.CallbackQuery.ID,
					fmt.Sprintf(replics.AnswerSuccessfullySubscribed, channelTitle),
				))
				continue
			}
		}
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "subscriptions":
					channels, err := db.GetChannelsByChat(int(update.Message.Chat.ID))
					if err != nil {
						log.Println(err)
						continue
					}
					var msgText = ""

					if len(channels) > 0 {
						msgText = replics.SubscriptionsHeader
						for _, channel := range channels {
							if channel != "" {
								msgText += "@" + channel + "\n"
							}
						}
						msgText += replics.SubscriptionsFooter
					} else {
						msgText = replics.SubscriptionsDefault
					}
					msg := tgbotapi.NewMessage(
						update.Message.Chat.ID,
						msgText,
					)
					bot.Send(msg)
				default:
					msg := tgbotapi.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf(replics.Help, bot.Self.FirstName),
					)
					_, err = bot.Send(msg)
					if err != nil {
						log.Println(err)
						continue
					}

				}
			} else if update.Message.ReplyToMessage != nil &&
				update.Message.ReplyToMessage.From.UserName == bot.Self.UserName &&
				update.Message.ReplyToMessage.Entities != nil {
				entities := *update.Message.ReplyToMessage.Entities
				var ent tgbotapi.MessageEntity
				if len(entities) == 0 {
					log.Println("Entities are not nil but empty. Passing..")
					continue
				} else {
					ent = entities[0]
				}
				urlParts := strings.Split(ent.URL, "/")
				if len(urlParts) < 4 {
					log.Println("URL from the entities is corrupt: ", ent.URL)
					continue
				}
				channelUsername := strings.Split(ent.URL, "/")[3]

				re := regexp.MustCompile("^Forwarded from (.*?)\\n\\n")
				parts := re.FindStringSubmatch(update.Message.ReplyToMessage.Text)
				var channelTitle string
				if len(parts) > 0 {
					channelTitle = parts[1]
				} else {
					channelTitle = channelUsername
				}

				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					fmt.Sprintf(replics.ReplyToSubscribed, channelTitle),
				)

				callbackData := fmt.Sprintf("unsubscribe-%s", channelUsername)
				msg.ReplyToMessageID = update.Message.ReplyToMessage.MessageID
				msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
						tgbotapi.InlineKeyboardButton{
							Text:         replics.ReplyButtonUnsubscribe,
							CallbackData: &callbackData,
						},
					}},
				}
				_, err = bot.Send(msg)
				if err != nil {
					log.Println(err)
				}
				continue
			} else if update.Message.ForwardFromChat != nil &&
				update.Message.ForwardFromChat.IsChannel() &&
				!update.Message.From.IsBot {
				if update.Message.Chat.ID != int64(unifeedBotToUserChatID) {
					var msgText string
					err = db.PutChatChannel(update.Message.From.ID, update.Message.ForwardFromChat.UserName)
					if err != nil {
						me, ok := err.(*mysql.MySQLError)
						if !ok {
							continue
						}
						if me.Number == 1062 {
							msgText = fmt.Sprintf(replics.AlreadySubscribed, update.Message.ForwardFromChat.Title, update.Message.ForwardFromChat.Title)
						} else {
							log.Println(err)
							continue
						}
					} else {
						msgText = fmt.Sprintf(
							replics.SuccessfullySubscribed,
							update.Message.ForwardFromChat.Title,
						)
					}
					ch <- update.Message.ForwardFromChat.UserName
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					msg.ParseMode = "markdown"
					_, err = bot.Send(msg)
					if err != nil {
						log.Println(err)
					}
					continue
				} else {
					chatIDs, err := db.GetChatsByChannel(update.Message.ForwardFromChat.UserName)
					if err != nil {
						log.Println(err)
						continue
					}

					var actualMessageText string = update.Message.Text

					if update.Message.Entities != nil {
						// Offsets of telegram entities are counted in UTF16 code units so we must encode them in UTF16 to properly count offsets and lengths
						// Problems appear on characters with >2 bytes
						TextInUTF16CodeUnits := utf16.Encode([]rune(update.Message.Text))
						var newMessageTextRunes []rune
						for i := range TextInUTF16CodeUnits {
							entityFound := false
							for _, entity := range *update.Message.Entities {
								if entity.Type == "text_link" {
									if i == entity.Offset {
										oldLinkTextUTF16 := TextInUTF16CodeUnits[entity.Offset : entity.Offset+entity.Length]
										oldLinkTextRunes := utf16.Decode(oldLinkTextUTF16)
										oldLinkTextString := string(oldLinkTextRunes)

										newLinkTextString := fmt.Sprintf("[%s](%s)", oldLinkTextString, entity.URL)
										newLinkTextRunes := []rune(newLinkTextString)

										for _, newRune := range newLinkTextRunes {
											newMessageTextRunes = append(newMessageTextRunes, newRune)
										}
										entityFound = true
									} else if i > entity.Offset && i < entity.Offset+entity.Length {
										entityFound = true
									}
								}
							}
							if !entityFound {
								var oldRune rune
								if utf16.IsSurrogate(rune(TextInUTF16CodeUnits[i])) {
									oldRune = utf16.DecodeRune(
										rune(TextInUTF16CodeUnits[i]),
										rune(TextInUTF16CodeUnits[i+1]),
									)
									if oldRune == 65533 { // Replacement character U+FFFD. Appears on the second code unit of a surrogate
										continue
									}
								} else {
									oldRune = utf16.Decode(TextInUTF16CodeUnits[i : i+1])[0]
								}
								newMessageTextRunes = append(newMessageTextRunes, oldRune)
							}
						}
						actualMessageText = string(newMessageTextRunes)
					}

					msgText := fmt.Sprintf(
						"[Forwarded from %s](https://t.me/%s/%d)",
						update.Message.ForwardFromChat.Title,
						update.Message.ForwardFromChat.UserName,
						update.Message.ForwardFromMessageID,
					)
					if actualMessageText != "" {
						msgText += "\n\n" + actualMessageText
					}
					for _, chatID := range chatIDs {
						msg := tgbotapi.NewMessage(
							int64(chatID),
							msgText,
						)
						msg.ParseMode = "markdown"
						_, err = bot.Send(msg)
						if err != nil {
							log.Println("Redistributing post error", err, "ChatID:", chatID)
						}
					}
				}
			}
		}
	}
}

func botLoop(ch chan string) {

	bot, err := tgbotapi.NewBotAPI(os.Getenv("UNIFEED_TELEGRAM_BOT_API_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authenticated on Telegram Bot account %s", bot.Self.UserName)

	debug := false
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" {
		debug = true
	}
	bot.Debug = debug

	bot.RemoveWebhook()
	var updates tgbotapi.UpdatesChannel
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err = bot.GetUpdatesChan(u)

	for w := 0; w < runtime.NumCPU()+2; w++ {
		go processBot(bot, updates, ch)
	}
	c := make(chan struct{})
	<-c
}

func subscriptionLoop(tdlibClient *client.Client, ch chan string) {
	for supergroupUsername := range ch {
		searchPublicChatsRequest := client.SearchPublicChatsRequest{
			Query: supergroupUsername,
		}
		chats, err := tdlibClient.SearchPublicChats(&searchPublicChatsRequest)
		if err != nil {
			log.Println(err)
			continue
		}
		if chats != nil && chats.TotalCount > 0 {
			chatID := chats.ChatIds[0]
			joinChatRequest := client.JoinChatRequest{
				ChatId: chatID,
			}
			ok, err := tdlibClient.JoinChat(&joinChatRequest)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("Subscription OK:%v for %s", ok, supergroupUsername)
		} else {
			log.Println("Empty SearchResult for searchChats")
		}
	}
}

func tdlibMain() {
	authorizer := client.ClientAuthorizer()
	go client.CliInteractor(authorizer)

	apiID, _ := strconv.Atoi(os.Getenv("UNIFEED_TELEGRAM_API_ID"))
	apiHash := os.Getenv("UNIFEED_TELEGRAM_API_HASH")

	authorizer.TdlibParameters <- &client.TdlibParameters{
		UseTestDc:              false,
		DatabaseDirectory:      filepath.Join(".tdlib", "database"),
		FilesDirectory:         filepath.Join(".tdlib", "files"),
		UseFileDatabase:        true,
		UseChatInfoDatabase:    true,
		UseMessageDatabase:     true,
		UseSecretChats:         false,
		ApiId:                  int32(apiID),
		ApiHash:                apiHash,
		SystemLanguageCode:     "en",
		DeviceModel:            "Server",
		SystemVersion:          "1.0.0",
		ApplicationVersion:     "1.0.0",
		EnableStorageOptimizer: true,
		IgnoreFileNames:        false,
	}

	logVerbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 0,
	})

	tdlibClient, err := client.NewClient(authorizer, logVerbosity)
	if err != nil {
		log.Fatalf("NewClient error: %s", err)
	}

	optionValue, err := tdlibClient.GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		log.Fatalf("GetOption error: %s", err)
	}
	log.Printf("TDLib version: %s", optionValue.(*client.OptionValueString).Value)

	me, err := tdlibClient.GetMe()
	if err != nil {
		log.Fatalf("GetMe error: %s", err)
	}
	log.Printf("Me: %s %s [%s]", me.FirstName, me.LastName, me.Username)

	listener := tdlibClient.GetListener()
	defer listener.Close()

	ch := make(chan string)
	go subscriptionLoop(tdlibClient, ch)
	go botLoop(ch)

	for update := range listener.Updates {
		if update.GetClass() == client.ClassUpdate {
			if update.GetType() == "updateNewMessage" {
				newMessageUpdate, ok := update.(*client.UpdateNewMessage)
				if !ok {
					continue
				}
				if newMessageUpdate.Message != nil && newMessageUpdate.Message.CanBeForwarded && newMessageUpdate.Message.IsChannelPost {
					log.Printf("Message from channel %#v. Id: %#v. Content Type: %s\n", newMessageUpdate.Message.ChatId, newMessageUpdate.Message.Id, newMessageUpdate.Message.Content.MessageContentType())
					unifeedUserToBotChatID, _ := strconv.Atoi(os.Getenv("UNIFEED_USER_TO_BOT_CHAT_ID"))
					fwdReq := &client.ForwardMessagesRequest{
						ChatId:     int64(unifeedUserToBotChatID),
						FromChatId: newMessageUpdate.Message.ChatId,
						MessageIds: []int64{newMessageUpdate.Message.Id},
					}
					_, err := tdlibClient.ForwardMessages(fwdReq)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
	}
}

func main() {
	defer db.DB.Close()

	var env string
	env = os.Getenv("UNIFEED_TELEGRAM_BOT_API_TOKEN")
	if env == "" {
		panic("NO UNIFEED_TELEGRAM_BOT_API_TOKEN")
	}

	env = os.Getenv("UNIFEED_TELEGRAM_API_ID")
	if env == "" {
		panic("NO UNIFEED_TELEGRAM_API_ID")
	}

	env = os.Getenv("UNIFEED_TELEGRAM_API_HASH")
	if env == "" {
		panic("NO UNIFEED_TELEGRAM_API_HASH")
	}

	env = os.Getenv("UNIFEED_USER_TO_BOT_CHAT_ID")
	if env == "" {
		panic("NO UNIFEED_USER_TO_BOT_CHAT_ID")
	}

	env = os.Getenv("UNIFEED_BOT_TO_USER_CHAT_ID")
	if env == "" {
		panic("NO UNIFEED_BOT_TO_USER_CHAT_ID")
	}

	env = os.Getenv("UNIFEED_SQL_DSN")
	if env == "" {
		panic("NO UNIFEED_SQL_DSN")
	}

	tdlibMain()
}
