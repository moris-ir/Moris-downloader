package media

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTranscodeAudioWithFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "in.wav")
	dst := filepath.Join(dir, "out.mp3")
	if err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", "0.2", src).Run(); err != nil {
		t.Fatal(err)
	}
	if err := transcode(context.Background(), src, dst, "audio", "mp3"); err != nil {
		t.Fatal(err)
	}
}
