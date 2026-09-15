package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Client struct {
	Token string
	HTTP  *http.Client
}

func New(token string) *Client {
	return &Client{Token: token, HTTP: &http.Client{Timeout: 15 * time.Minute}}
}

type apiResponse struct {
	OK          bool
	Result      json.RawMessage
	Description string
}

func (c *Client) call(ctx context.Context, method string, vals url.Values) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.Token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(vals.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if !ar.OK {
		return nil, fmt.Errorf("telegram: %s", ar.Description)
	}
	return ar.Result, nil
}
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := c.call(ctx, "sendMessage", url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}, "text": {text}})
	return err
}

// SendFile uploads a local file to a chat using Telegram's multipart Bot API.
func (c *Client) SendFile(ctx context.Context, chatID int64, path, field, caption string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer pw.Close()
		part, e := mw.CreateFormFile(field, filepath.Base(path))
		if e != nil {
			_ = pw.CloseWithError(e)
			done <- e
			return
		}
		if _, e = io.Copy(part, f); e != nil {
			_ = pw.CloseWithError(e)
			done <- e
			return
		}
		if e = mw.WriteField("chat_id", strconv.FormatInt(chatID, 10)); e != nil {
			_ = pw.CloseWithError(e)
			done <- e
			return
		}
		if caption != "" {
			if e = mw.WriteField("caption", caption); e != nil {
				_ = pw.CloseWithError(e)
				done <- e
				return
			}
		}
		if e = mw.Close(); e != nil {
			_ = pw.CloseWithError(e)
			done <- e
			return
		}
		done <- nil
	}()
	method := "sendDocument"
	if field == "video" {
		method = "sendVideo"
	} else if field == "audio" {
		method = "sendAudio"
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.Token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var ar apiResponse
	if err = json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return err
	}
	if err = <-done; err != nil {
		return err
	}
	if !ar.OK {
		return fmt.Errorf("telegram: %s", ar.Description)
	}
	return nil
}
