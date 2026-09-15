package music

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Track struct {
	Artist      string `json:"artist"`
	Title       string `json:"title"`
	Album       string `json:"album,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
	Link        string `json:"link,omitempty"`
	Artwork     string `json:"artwork,omitempty"`
}
type Lyrics struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Plain  string `json:"plain,omitempty"`
	Synced string `json:"synced,omitempty"`
	Source string `json:"source"`
}

type Recognizer interface {
	Recognize(context.Context, string) (Track, error)
}
type LyricsProvider interface {
	Search(context.Context, string, string) (Lyrics, error)
}

type AudD struct {
	Token string
	HTTP  *http.Client
}

func NewAudD(token string) *AudD {
	return &AudD{Token: token, HTTP: &http.Client{Timeout: 2 * time.Minute}}
}

type auddResponse struct {
	Status string `json:"status"`
	Result struct {
		Artist      string `json:"artist"`
		Title       string `json:"title"`
		Album       string `json:"album"`
		ReleaseDate string `json:"release_date"`
		SongLink    string `json:"song_link"`
		AppleMusic  struct {
			Artwork string `json:"artwork"`
		} `json:"apple_music"`
	} `json:"result"`
	Error string `json:"error"`
}

func (a *AudD) Recognize(ctx context.Context, path string) (Track, error) {
	if a.Token == "" {
		return Track{}, errors.New("music recognition token is not configured")
	}
	f, e := os.Open(path)
	if e != nil {
		return Track{}, e
	}
	defer f.Close()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("api_token", a.Token)
	part, e := mw.CreateFormFile("file", path)
	if e != nil {
		return Track{}, e
	}
	if _, e = io.Copy(part, f); e != nil {
		return Track{}, e
	}
	if e = mw.Close(); e != nil {
		return Track{}, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.audd.io/", &b)
	if e != nil {
		return Track{}, e
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, e := a.HTTP.Do(req)
	if e != nil {
		return Track{}, e
	}
	defer resp.Body.Close()
	var out auddResponse
	if e = json.NewDecoder(resp.Body).Decode(&out); e != nil {
		return Track{}, e
	}
	if resp.StatusCode >= 300 || out.Status != "success" {
		return Track{}, fmt.Errorf("audd recognition failed: %s", out.Error)
	}
	return Track{Artist: out.Result.Artist, Title: out.Result.Title, Album: out.Result.Album, ReleaseDate: out.Result.ReleaseDate, Link: out.Result.SongLink, Artwork: out.Result.AppleMusic.Artwork}, nil
}

type LRCLIB struct{ HTTP *http.Client }

func NewLRCLIB() *LRCLIB { return &LRCLIB{HTTP: &http.Client{Timeout: 15 * time.Second}} }

type lrcResponse struct {
	TrackName    string `json:"trackName"`
	ArtistName   string `json:"artistName"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}

func (l *LRCLIB) Search(ctx context.Context, artist, title string) (Lyrics, error) {
	u := "https://lrclib.net/api/get?artist_name=" + url.QueryEscape(artist) + "&track_name=" + url.QueryEscape(title)
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if e != nil {
		return Lyrics{}, e
	}
	resp, e := l.HTTP.Do(req)
	if e != nil {
		return Lyrics{}, e
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return Lyrics{}, errors.New("lyrics not found")
	}
	if resp.StatusCode >= 300 {
		return Lyrics{}, fmt.Errorf("lyrics provider: %s", resp.Status)
	}
	var out lrcResponse
	if e = json.NewDecoder(resp.Body).Decode(&out); e != nil {
		return Lyrics{}, e
	}
	return Lyrics{Artist: out.ArtistName, Title: out.TrackName, Plain: strings.TrimSpace(out.PlainLyrics), Synced: strings.TrimSpace(out.SyncedLyrics), Source: "LRCLIB"}, nil
}
