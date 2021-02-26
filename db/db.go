package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func init() {
	var err error
	dsn := os.Getenv("UNIFEED_SQL_DSN")
	if dsn != "" {
		DB, err = sql.Open(
			"mysql",
			dsn,
		)
		if err != nil {
			log.Println(err)
		}
		err = DB.Ping()
		if err != nil {
			log.Fatal(err)
		}
	}
}

type Entity struct {
	ID              string
	ChatID          int
	ChannelUsername string
}

func GetChatsByChannel(channelUsername string) ([]int, error) {
	var err error
	var result []int

	if DB == nil {
		return nil, nil
	}
	resp, err := DB.Query("SELECT DISTINCT chat_id FROM chat_has_channel WHERE channel_username=?", channelUsername)
	if err != nil {
		return nil, err
	}
	for resp.Next() {
		var chatID int
		resp.Scan(&chatID)
		result = append(result, chatID)
	}
	return result, err
}

func GetChannelsByChat(chatID int) ([]string, error) {
	var err error
	var result []string

	if DB == nil {
		return nil, nil
	}
	resp, err := DB.Query("SELECT DISTINCT channel_username FROM chat_has_channel WHERE chat_id=?", chatID)
	if err != nil {
		return nil, err
	}
	for resp.Next() {
		var channelUsername string
		resp.Scan(&channelUsername)
		result = append(result, channelUsername)
	}
	return result, err
}

func PutChatChannel(chatID int, channelUsername string) error {
	var err error
	id := fmt.Sprintf("%d@%s", chatID, channelUsername)
	_, err = DB.Exec(
		"INSERT INTO chat_has_channel (id, chat_id, channel_username) VALUES (?, ?, ?)",
		id,
		chatID,
		channelUsername,
	)
	return err
}

func DeleteChatChannel(chatID int, channelUsername string) error {
	var err error
	id := fmt.Sprintf("%d@%s", chatID, channelUsername)
	_, err = DB.Exec(
		"DELETE FROM chat_has_channel WHERE id=?",
		id,
	)
	return err
}
