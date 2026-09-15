package media

import "testing"

func TestValidateURL(t *testing.T) {
	if ValidateURL("https://example.com/x") != nil {
		t.Fatal("expected valid")
	}
	if ValidateURL("javascript:alert(1)") == nil {
		t.Fatal("expected invalid")
	}
}
func TestDetectPlatform(t *testing.T) {
	cases := map[string]string{"https://youtube.com/watch?v=x": "youtube", "https://instagram.com/reel/x": "instagram", "https://tiktok.com/@x/video/1": "tiktok", "https://foo.example/x": "generic"}
	for u, w := range cases {
		if got := DetectPlatform(u); got != w {
			t.Fatalf("%s: got %s want %s", u, got, w)
		}
	}
}
