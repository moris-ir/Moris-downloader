package media

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/moris3245/moris/internal/domain"
)

type Engine struct {
	WorkDir  string
	MaxBytes int64
}

func NewEngine(dir string, max int64) *Engine { return &Engine{WorkDir: dir, MaxBytes: max} }

var urlRE = regexp.MustCompile(`^https?://[^\s]+$`)

func ValidateURL(u string) error {
	if !urlRE.MatchString(u) {
		return errors.New("invalid URL")
	}
	return nil
}
func DetectPlatform(u string) string {
	l := strings.ToLower(u)
	switch {
	case strings.Contains(l, "youtube.com"), strings.Contains(l, "youtu.be"):
		return "youtube"
	case strings.Contains(l, "instagram.com"):
		return "instagram"
	case strings.Contains(l, "tiktok.com"):
		return "tiktok"
	case strings.Contains(l, "twitch.tv"):
		return "twitch"
	case strings.Contains(l, "kick.com"):
		return "kick"
	case strings.Contains(l, "x.com"), strings.Contains(l, "twitter.com"):
		return "x"
	case strings.Contains(l, "facebook.com"), strings.Contains(l, "fb.watch"):
		return "facebook"
	case strings.Contains(l, "reddit.com"):
		return "reddit"
	case strings.Contains(l, "pinterest.com"):
		return "pinterest"
	case strings.Contains(l, "soundcloud.com"):
		return "soundcloud"
	case strings.Contains(l, "vimeo.com"):
		return "vimeo"
	default:
		return "generic"
	}
}

type rawFormat struct {
	FormatID, Ext, Vcodec, Acodec string
	Width, Height                 int
	FPS, Tbr                      float64
	Filesize, FilesizeApprox      int64
}
type rawInfo struct {
	Title, Uploader, Thumbnail, WebpageURL string
	Duration                               int64
	Vcodec, Acodec                         string
	Formats                                []rawFormat
}

func (e *Engine) Analyze(ctx context.Context, u string) (domain.MediaInfo, error) {
	if err := ValidateURL(u); err != nil {
		return domain.MediaInfo{}, err
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return domain.MediaInfo{URL: u, Platform: DetectPlatform(u), ContentType: "unknown", Title: "Metadata unavailable (yt-dlp missing)"}, nil
	}
	cmd := exec.CommandContext(ctx, "yt-dlp", "--dump-single-json", "--no-playlist", "--skip-download", "--no-warnings", "--", ""+u)
	out, err := cmd.Output()
	if err != nil {
		return domain.MediaInfo{}, fmt.Errorf("metadata: %w", err)
	}
	var raw rawInfo
	if err = json.Unmarshal(out, &raw); err != nil {
		return domain.MediaInfo{}, err
	}
	mi := domain.MediaInfo{URL: u, Platform: DetectPlatform(u), ContentType: contentType(raw.Vcodec, raw.Acodec), Title: raw.Title, Uploader: raw.Uploader, Thumbnail: raw.Thumbnail, Duration: raw.Duration}
	for _, f := range raw.Formats {
		size := f.Filesize
		if size == 0 {
			size = f.FilesizeApprox
		}
		typ := "video"
		if f.Vcodec == "none" {
			typ = "audio"
		}
		mi.Formats = append(mi.Formats, domain.Format{ID: f.FormatID, Type: typ, Container: f.Ext, Codec: codec(f.Vcodec, f.Acodec), Width: f.Width, Height: f.Height, FPS: f.FPS, Bitrate: int64(f.Tbr), FileSize: size})
	}
	sort.Slice(mi.Formats, func(i, j int) bool { return mi.Formats[i].Height > mi.Formats[j].Height })
	return mi, nil
}
func contentType(v, a string) string {
	if v == "none" || v == "" {
		return "audio"
	}
	if a == "none" {
		return "video"
	}
	return "video"
}
func codec(v, a string) string {
	if v == "none" {
		return a
	}
	if a == "none" {
		return v
	}
	return v + "/" + a
}
func normalizeQuality(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	switch q {
	case "best", "1080", "720", "480", "360":
		return q
	default:
		return "best"
	}
}
func (e *Engine) Download(ctx context.Context, j domain.Job) (domain.DownloadResult, error) {
	if err := ValidateURL(j.URL); err != nil {
		return domain.DownloadResult{}, err
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return domain.DownloadResult{}, errors.New("yt-dlp is not installed")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return domain.DownloadResult{}, errors.New("ffmpeg is not installed")
	}
	dir := filepath.Join(e.WorkDir, "jobs", j.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return domain.DownloadResult{}, err
	}
	outtmpl := filepath.Join(dir, "source.%(ext)s")
	format := j.Format
	quality := normalizeQuality(j.Quality)
	if format == "" {
		if j.Type == domain.JobAudio {
			format = "m4a"
		} else {
			format = "mp4"
		}
	}
	args := []string{"--no-playlist", "--newline", "--no-warnings", "-o", outtmpl}
	if j.Type == domain.JobAudio {
		args = append(args, "-f", audioSelector(j.Format), "--extract-audio", "--audio-format", audioExt(j.Format))
	} else {
		args = append(args, "-f", videoSelector(quality, j.Format), "--merge-output-format", container(j.Format))
	}
	args = append(args, "--", j.URL)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return domain.DownloadResult{}, err
	}
	if err = cmd.Start(); err != nil {
		return domain.DownloadResult{}, fmt.Errorf("start yt-dlp: %w", err)
	}
	s := bufio.NewScanner(stdout)
	for s.Scan() {
	}
	if err = cmd.Wait(); err != nil {
		return domain.DownloadResult{}, fmt.Errorf("download: %w", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return domain.DownloadResult{}, err
	}
	var p string
	var newest time.Time
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, er := f.Info()
		if er != nil {
			continue
		}
		if filepath.Ext(f.Name()) == ".part" || filepath.Ext(f.Name()) == ".ytdl" {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
			p = filepath.Join(dir, f.Name())
		}
	}
	if p == "" {
		return domain.DownloadResult{}, errors.New("yt-dlp produced no file")
	}
	// Explicit FFmpeg normalization guarantees the requested output container/codec when possible.
	ext := strings.ToLower(filepath.Ext(p))
	want := strings.ToLower("." + outputExt(j.Type, format))
	if want != "." && ext != want {
		dst := filepath.Join(dir, "final."+strings.TrimPrefix(want, "."))
		if err := transcode(ctx, p, dst, j.Type, format); err != nil {
			return domain.DownloadResult{}, err
		}
		p = dst
	}
	st, err := os.Stat(p)
	if err != nil {
		return domain.DownloadResult{}, err
	}
	if e.MaxBytes > 0 && st.Size() > e.MaxBytes {
		return domain.DownloadResult{}, fmt.Errorf("file exceeds configured limit: %d bytes", st.Size())
	}
	return domain.DownloadResult{Path: p, Size: st.Size(), ContentType: mimeFor(p, j.Type)}, nil
}
func videoSelector(q, format string) string {
	base := "bestvideo*+bestaudio/best"
	if q != "best" {
		base = "bestvideo[height<=" + q + "]*+bestaudio/best[height<=" + q + "]"
	}
	if strings.EqualFold(format, "webm") {
		return base + "[ext=webm]/" + base
	}
	return base
}
func audioSelector(format string) string {
	if strings.EqualFold(format, "mp3") || strings.EqualFold(format, "wav") {
		return "bestaudio/best"
	}
	return "bestaudio/best"
}
func audioExt(format string) string {
	switch strings.ToLower(format) {
	case "mp3", "wav", "m4a", "opus", "flac":
		return strings.ToLower(format)
	default:
		return "m4a"
	}
}
func container(format string) string {
	switch strings.ToLower(format) {
	case "webm", "mkv", "mp4":
		return strings.ToLower(format)
	default:
		return "mp4"
	}
}
func outputExt(t domain.JobType, f string) string {
	if t == domain.JobAudio {
		return audioExt(f)
	}
	return container(f)
}
func mimeFor(p string, t domain.JobType) string {
	if t == domain.JobAudio {
		switch strings.ToLower(filepath.Ext(p)) {
		case ".mp3":
			return "audio/mpeg"
		case ".m4a":
			return "audio/mp4"
		case ".wav":
			return "audio/wav"
		case ".flac":
			return "audio/flac"
		}
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	}
	return "application/octet-stream"
}
func transcode(ctx context.Context, src, dst string, t domain.JobType, format string) error {
	args := []string{"-y", "-i", src}
	if t == domain.JobAudio {
		switch strings.ToLower(format) {
		case "mp3":
			args = append(args, "-vn", "-codec:a", "libmp3lame", "-b:a", "320k")
		case "wav":
			args = append(args, "-vn", "-codec:a", "pcm_s16le")
		case "flac":
			args = append(args, "-vn", "-codec:a", "flac")
		default:
			args = append(args, "-vn", "-codec:a", "aac", "-b:a", "256k")
		}
	} else {
		switch strings.ToLower(format) {
		case "webm":
			args = append(args, "-c:v", "libvpx-vp9", "-c:a", "libopus")
		case "mkv":
			args = append(args, "-c", "copy")
		default:
			args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-c:a", "aac", "-b:a", "192k")
		}
	}
	args = append(args, dst)
	return exec.CommandContext(ctx, "ffmpeg", args...).Run()
}
func (e *Engine) Cleanup(maxAge time.Duration) error {
	root := filepath.Join(e.WorkDir, "jobs")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	cut := time.Now().Add(-maxAge)
	for _, x := range entries {
		if !x.IsDir() {
			continue
		}
		p := filepath.Join(root, x.Name())
		st, er := os.Stat(p)
		if er == nil && st.ModTime().Before(cut) {
			_ = os.RemoveAll(p)
		}
	}
	return nil
}
func (e *Engine) String() string { return strconv.FormatInt(e.MaxBytes, 10) }
