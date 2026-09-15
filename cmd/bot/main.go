package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/moris3245/moris/internal/config"
	"github.com/moris3245/moris/internal/telegram"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var pending = struct {
	sync.RWMutex
	m map[int64]string
}{m: map[int64]string{}}

type apiClient struct {
	base, token string
	http        *http.Client
}

func (c apiClient) createJob(ctx context.Context, userID, url, jtype string, quality ...string) (map[string]any, error) {
	payload := map[string]string{"user_id": userID, "url": url, "type": jtype}
	if len(quality) > 0 {
		if jtype == "audio" {
			payload["format"] = quality[0]
		} else {
			payload["quality"] = quality[0]
		}
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/jobs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MORIS-BOT-TOKEN", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api %s: %s", resp.Status, string(raw))
	}
	var out map[string]any
	return out, json.Unmarshal(raw, &out)
}

func (c apiClient) analyze(ctx context.Context, rawURL string) (map[string]any, error) {
	b, _ := json.Marshal(map[string]string{"url": rawURL})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/media/analyze", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MORIS-BOT-TOKEN", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api %s: %s", resp.Status, string(raw))
	}
	var out map[string]any
	return out, json.Unmarshal(raw, &out)
}

func (c apiClient) getJob(ctx context.Context, id string) (map[string]any, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/jobs/"+id, nil)
	req.Header.Set("X-MORIS-BOT-TOKEN", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func main() {
	cfg := config.Load()
	if cfg.TelegramToken == "" {
		log.Println("MORIS bot: token missing")
		return
	}
	apiBase := os.Getenv("MORIS_PUBLIC_BASE_URL")
	if apiBase == "" {
		apiBase = "http://api:8080"
	}
	tg := telegram.New(cfg.TelegramToken)
	api := apiClient{base: strings.TrimRight(apiBase, "/"), token: cfg.AdminToken, http: &http.Client{Timeout: 20 * time.Second}}
	log.Println("MORIS Telegram bot polling started")
	err := tg.Poll(context.Background(), func(ctx context.Context, u telegram.Update) error {
		if u.CallbackQuery != nil {
			return handleCallback(ctx, tg, api, u)
		}
		if u.Message == nil {
			return nil
		}
		chatID := u.Message.Chat.ID
		text := strings.TrimSpace(u.Message.Text)
		if text == "/start" || text == "/help" {
			return tg.SendMenu(ctx, chatID, "MORIS — Universal Media Intelligence Bot\n\nلینک ویدیو/صوت را ارسال کنید تا پردازش شود.\n\nCommands: /start /help /status <job_id>")
		}
		if strings.HasPrefix(text, "/status ") {
			id := strings.TrimSpace(strings.TrimPrefix(text, "/status "))
			j, e := api.getJob(ctx, id)
			if e != nil {
				return tg.SendMessage(ctx, chatID, "❌ خطا در دریافت وضعیت: "+e.Error())
			}
			return tg.SendMessage(ctx, chatID, fmt.Sprintf("Job %s\nStatus: %v\nProgress: %v%%\nError: %v", id, j["status"], j["progress"], j["error"]))
		}
		uurl := telegram.ExtractURL(text)
		if uurl == "" {
			return tg.SendMenu(ctx, chatID, "یک URL معتبر ارسال کنید یا /help را بزنید.")
		}
		mi, e := api.analyze(ctx, uurl)
		if e != nil {
			return tg.SendMessage(ctx, chatID, "❌ تحلیل لینک ناموفق بود: "+e.Error())
		}
		pending.Lock()
		pending.m[chatID] = uurl
		pending.Unlock()
		title := fmt.Sprintf("%v", mi["title"])
		if title == "" || title == "<nil>" {
			title = "Media"
		}
		return tg.SendQualityMenu(ctx, chatID, "🔎 "+title+"\nPlatform: "+fmt.Sprintf("%v", mi["platform"])+"\n\nکیفیت/نوع خروجی را انتخاب کنید:")
	})
	if err != nil {
		log.Fatal(err)
	}
}
func handleCallback(ctx context.Context, tg *telegram.Client, api apiClient, u telegram.Update) error {
	q := u.CallbackQuery
	if q == nil {
		return nil
	}
	_ = tg.AnswerCallback(ctx, q.ID, "")
	if q.Message == nil {
		return nil
	}
	switch {
	case q.Data == "type:video":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "🎬 لینک را ارسال کنید تا کیفیت و فرمت پویا نمایش داده شود.")
	case q.Data == "type:audio":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "🎵 لینک را ارسال کنید تا فرمت صوتی را انتخاب کنید.")
	case strings.HasPrefix(q.Data, "video:") || strings.HasPrefix(q.Data, "audio:"):
		pending.RLock()
		url := pending.m[q.Message.Chat.ID]
		pending.RUnlock()
		if url == "" {
			return tg.SendMessage(ctx, q.Message.Chat.ID, "⚠️ لینک منقضی شده؛ دوباره لینک را ارسال کنید.")
		}
		parts := strings.SplitN(q.Data, ":", 2)
		typ, quality := parts[0], parts[1]
		job, e := api.createJob(ctx, strconv.FormatInt(q.Message.Chat.ID, 10), url, typ, quality)
		if e != nil {
			return tg.SendMessage(ctx, q.Message.Chat.ID, "❌ ثبت Job ناموفق بود: "+e.Error())
		}
		pending.Lock()
		delete(pending.m, q.Message.Chat.ID)
		pending.Unlock()
		return tg.SendMessage(ctx, q.Message.Chat.ID, fmt.Sprintf("✅ Job ثبت شد\nID: %v\nQuality: %s\n\n/status %v", job["id"], quality, job["id"]))
	case q.Data == "music":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "🎧 Music Intelligence provider در حال آماده‌سازی است.")
	case q.Data == "history":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "📜 History در پنل API/Admin قابل مشاهده است.")
	case q.Data == "vip":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "💎 VIP plans در نسخه پرداخت فعال می‌شود.")
	case q.Data == "language":
		return tg.SendMessage(ctx, q.Message.Chat.ID, "🌐 زبان‌ها: فارسی / English / Deutsch / العربية / Español")
	default:
		return nil
	}
}
