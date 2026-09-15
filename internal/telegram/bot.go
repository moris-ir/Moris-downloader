package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text,omitempty"`
}
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}
type Chat struct {
	ID int64 `json:"id"`
}
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from,omitempty"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type getUpdatesResult struct {
	Result []Update `json:"result"`
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	vals := url.Values{"offset": {strconv.FormatInt(offset, 10)}, "timeout": {strconv.Itoa(timeout)}, "allowed_updates": {`["message","callback_query"]`}}
	raw, err := c.call(ctx, "getUpdates", vals)
	if err != nil {
		return nil, err
	}
	var r getUpdatesResult
	if err = json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return r.Result, nil
}
func (c *Client) AnswerCallback(ctx context.Context, id, text string) error {
	_, err := c.call(ctx, "answerCallbackQuery", url.Values{"callback_query_id": {id}, "text": {text}})
	return err
}
func (c *Client) SendMenu(ctx context.Context, chatID int64, text string) error {
	b, _ := json.Marshal(InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "🎬 Video", CallbackData: "type:video"}, {Text: "🎵 Audio", CallbackData: "type:audio"}}, {{Text: "🎧 Music Recognition", CallbackData: "music"}}, {{Text: "📜 History", CallbackData: "history"}, {Text: "💎 VIP", CallbackData: "vip"}}, {{Text: "🌐 Language", CallbackData: "language"}}}})
	_, err := c.call(ctx, "sendMessage", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "text": {text}, "reply_markup": {string(b)}})
	return err
}
func (c *Client) SendQualityMenu(ctx context.Context, chatID int64, text string) error {
	b, _ := json.Marshal(InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{
		{{Text: "🎬 Best", CallbackData: "video:best"}, {Text: "1080p", CallbackData: "video:1080"}, {Text: "720p", CallbackData: "video:720"}},
		{{Text: "480p", CallbackData: "video:480"}, {Text: "360p", CallbackData: "video:360"}},
		{{Text: "🎵 MP3", CallbackData: "audio:mp3"}, {Text: "M4A", CallbackData: "audio:m4a"}, {Text: "WAV", CallbackData: "audio:wav"}},
	}})
	_, err := c.call(ctx, "sendMessage", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "text": {text}, "reply_markup": {string(b)}})
	return err
}

func (c *Client) EditMessage(ctx context.Context, chatID, msgID int64, text string) error {
	_, err := c.call(ctx, "editMessageText", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "message_id": {strconv.FormatInt(msgID, 10)}, "text": {text}})
	return err
}
func (c *Client) SendChatAction(ctx context.Context, chatID int64, action string) error {
	_, err := c.call(ctx, "sendChatAction", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "action": {action}})
	return err
}
func (c *Client) Poll(ctx context.Context, handler func(context.Context, Update) error) error {
	var offset int64
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		ups, err := c.GetUpdates(ctx, offset, 30)
		if err != nil {
			select {
			case <-time.After(backoff):
			default:
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for _, u := range ups {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if err := handler(ctx, u); err != nil {
				fmt.Printf("telegram update %d: %v\n", u.UpdateID, err)
			}
		}
	}
}
func ExtractURL(text string) string {
	for _, f := range strings.Fields(text) {
		if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
			return strings.Trim(f, "<>()[]{}\"'.,")
		}
	}
	return ""
}
