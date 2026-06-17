package llm

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// 多模态消息构造便利函数。
//
// 多模态是「消息级」而非「模型级」属性，因此这些 helper 直接构造
// *schema.Message，无需改动 ModelConfig。底层使用 eino v0.9 的
// UserInputMultiContent（旧的 MultiContent 字段已被 eino 标记 Deprecated）。
//
// URL 仅允许 http / https / data（base64 data URI）三种协议；其它（如 file://）
// 会返回 ErrUnsupportedContentURLScheme，以防部分 Provider 客户端抓取时被诱导
// 读取本地文件或访问内网（SSRF 纵深防御）。模型是否真正支持多模态取决于
// 具体 Provider（GPT-4o / Claude 3+ / Gemini 等）。

// allowedContentURLSchemes 是多模态内容 URL 允许的协议白名单。
var allowedContentURLSchemes = map[string]bool{"http": true, "https": true, "data": true}

// UserImageMessage 构造一条「图片 + 文本」用户消息。text 为空时仅含图片。
func UserImageMessage(imageURL, text string) (*schema.Message, error) {
	if err := validateContentURL(imageURL); err != nil {
		return nil, err
	}
	parts := appendTextPart([]schema.MessageInputPart{imageInputPart(imageURL)}, text)
	return userMultiContentMessage(parts), nil
}

// UserImageMessages 构造一条「多图 + 文本」用户消息，图片顺序保留。
func UserImageMessages(imageURLs []string, text string) (*schema.Message, error) {
	parts := make([]schema.MessageInputPart, 0, len(imageURLs)+1)
	for _, u := range imageURLs {
		if err := validateContentURL(u); err != nil {
			return nil, err
		}
		parts = append(parts, imageInputPart(u))
	}
	return userMultiContentMessage(appendTextPart(parts, text)), nil
}

// UserAudioMessage 构造一条「音频 + 文本」用户消息（如 Gemini / GPT-4o-audio）。
func UserAudioMessage(audioURL, text string) (*schema.Message, error) {
	if err := validateContentURL(audioURL); err != nil {
		return nil, err
	}
	part := schema.MessageInputPart{
		Type:  schema.ChatMessagePartTypeAudioURL,
		Audio: &schema.MessageInputAudio{MessagePartCommon: urlCommon(audioURL)},
	}
	return userMultiContentMessage(appendTextPart([]schema.MessageInputPart{part}, text)), nil
}

// validateContentURL 校验内容 URL 协议在白名单内（http/https/data）。
func validateContentURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %q: %v", ErrUnsupportedContentURLScheme, raw, err)
	}
	if !allowedContentURLSchemes[strings.ToLower(u.Scheme)] {
		return fmt.Errorf("%w: %q (allowed: http/https/data)", ErrUnsupportedContentURLScheme, u.Scheme)
	}
	return nil
}

func imageInputPart(u string) schema.MessageInputPart {
	return schema.MessageInputPart{
		Type:  schema.ChatMessagePartTypeImageURL,
		Image: &schema.MessageInputImage{MessagePartCommon: urlCommon(u)},
	}
}

func appendTextPart(parts []schema.MessageInputPart, text string) []schema.MessageInputPart {
	if text == "" {
		return parts
	}
	return append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
}

func urlCommon(u string) schema.MessagePartCommon {
	return schema.MessagePartCommon{URL: &u}
}

func userMultiContentMessage(parts []schema.MessageInputPart) *schema.Message {
	return &schema.Message{Role: schema.User, UserInputMultiContent: parts}
}
