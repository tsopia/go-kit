package llm

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestUserImageMessage(t *testing.T) {
	msg := UserImageMessage("https://example.com/cat.png", "这是什么动物？")

	if msg.Role != schema.User {
		t.Fatalf("role = %v, want User", msg.Role)
	}
	if len(msg.UserInputMultiContent) != 2 {
		t.Fatalf("parts = %d, want 2 (image + text)", len(msg.UserInputMultiContent))
	}
	img := msg.UserInputMultiContent[0]
	if img.Type != schema.ChatMessagePartTypeImageURL {
		t.Errorf("part[0].Type = %v, want image_url", img.Type)
	}
	if img.Image == nil || img.Image.URL == nil || *img.Image.URL != "https://example.com/cat.png" {
		t.Errorf("part[0] image url not set correctly: %+v", img.Image)
	}
	txt := msg.UserInputMultiContent[1]
	if txt.Type != schema.ChatMessagePartTypeText || txt.Text != "这是什么动物？" {
		t.Errorf("part[1] text not set correctly: %+v", txt)
	}
}

func TestUserImageMessage_EmptyTextOmitsTextPart(t *testing.T) {
	msg := UserImageMessage("https://example.com/cat.png", "")
	if len(msg.UserInputMultiContent) != 1 {
		t.Fatalf("parts = %d, want 1 (image only)", len(msg.UserInputMultiContent))
	}
	if msg.UserInputMultiContent[0].Type != schema.ChatMessagePartTypeImageURL {
		t.Errorf("expected only image part")
	}
}

func TestUserImageMessages_PreservesOrder(t *testing.T) {
	urls := []string{"https://a.png", "https://b.png", "https://c.png"}
	msg := UserImageMessages(urls, "compare")

	if len(msg.UserInputMultiContent) != 4 {
		t.Fatalf("parts = %d, want 4 (3 images + text)", len(msg.UserInputMultiContent))
	}
	for i, u := range urls {
		p := msg.UserInputMultiContent[i]
		if p.Type != schema.ChatMessagePartTypeImageURL {
			t.Errorf("part[%d].Type = %v, want image_url", i, p.Type)
		}
		if p.Image == nil || p.Image.URL == nil || *p.Image.URL != u {
			t.Errorf("part[%d] url = %v, want %q", i, p.Image, u)
		}
	}
	last := msg.UserInputMultiContent[3]
	if last.Type != schema.ChatMessagePartTypeText || last.Text != "compare" {
		t.Errorf("last part should be text 'compare', got %+v", last)
	}
}

func TestUserAudioMessage(t *testing.T) {
	msg := UserAudioMessage("https://example.com/a.wav", "转写")
	if len(msg.UserInputMultiContent) != 2 {
		t.Fatalf("parts = %d, want 2", len(msg.UserInputMultiContent))
	}
	a := msg.UserInputMultiContent[0]
	if a.Type != schema.ChatMessagePartTypeAudioURL {
		t.Errorf("part[0].Type = %v, want audio_url", a.Type)
	}
	if a.Audio == nil || a.Audio.URL == nil || *a.Audio.URL != "https://example.com/a.wav" {
		t.Errorf("audio url not set: %+v", a.Audio)
	}
}
