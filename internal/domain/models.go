package domain

import "time"

type UserStatus string

const (
	UserActive UserStatus = "active"
	UserBanned UserStatus = "banned"
)

type Plan string

const (
	PlanFree    Plan = "free"
	PlanVIP     Plan = "vip"
	PlanPremium Plan = "premium"
	PlanCustom  Plan = "custom"
)

type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobQueued     JobStatus = "queued"
	JobProcessing JobStatus = "processing"
	JobUploading  JobStatus = "uploading"
	JobCompleted  JobStatus = "completed"
	JobFailed     JobStatus = "failed"
	JobCancelled  JobStatus = "cancelled"
)

type JobType string

const (
	JobVideo JobType = "video"
	JobAudio JobType = "audio"
)

type User struct {
	ID                  string     `json:"id"`
	TelegramID          int64      `json:"telegram_id"`
	Username            string     `json:"username,omitempty"`
	FirstName           string     `json:"first_name,omitempty"`
	Language            string     `json:"language,omitempty"`
	RegisteredAt        time.Time  `json:"registered_at"`
	LastActivityAt      time.Time  `json:"last_activity_at"`
	DownloadCount       int64      `json:"download_count"`
	FailedJobs          int64      `json:"failed_jobs"`
	StorageUsage        int64      `json:"storage_usage"`
	Status              UserStatus `json:"status"`
	Plan                Plan       `json:"plan"`
	DefaultVideoQuality string     `json:"default_video_quality,omitempty"`
	DefaultAudioQuality string     `json:"default_audio_quality,omitempty"`
	PreferredFormat     string     `json:"preferred_format,omitempty"`
	Notifications       bool       `json:"notifications"`
	AutoDownload        bool       `json:"auto_download"`
}
type MediaInfo struct {
	URL         string   `json:"url"`
	Platform    string   `json:"platform"`
	ContentType string   `json:"content_type"`
	Title       string   `json:"title"`
	Uploader    string   `json:"uploader,omitempty"`
	Thumbnail   string   `json:"thumbnail,omitempty"`
	Duration    int64    `json:"duration"`
	Formats     []Format `json:"formats,omitempty"`
}
type Format struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Container string  `json:"container"`
	Codec     string  `json:"codec"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	FPS       float64 `json:"fps,omitempty"`
	Bitrate   int64   `json:"bitrate,omitempty"`
	FileSize  int64   `json:"file_size,omitempty"`
}
type Job struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	URL            string    `json:"url"`
	Format         string    `json:"format"`
	Quality        string    `json:"quality"`
	Type           JobType   `json:"type"`
	Status         JobStatus `json:"status"`
	Progress       float64   `json:"progress"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	Attempts       int       `json:"attempts"`
	TelegramChatID int64     `json:"telegram_chat_id,omitempty"`
	OutputPath     string    `json:"output_path,omitempty"`
	OutputSize     int64     `json:"output_size,omitempty"`
}
type DownloadResult struct {
	Path        string
	Size        int64
	ContentType string
}
