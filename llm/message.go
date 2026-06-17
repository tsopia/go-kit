package llm

import "github.com/cloudwego/eino/schema"

// 多模态消息构造便利函数。
//
// 多模态是「消息级」而非「模型级」属性，因此这些 helper 直接构造
// *schema.Message，无需改动 ModelConfig。底层使用 eino v0.9 的
// UserInputMultiContent（旧的 MultiContent 字段已被 eino 标记 Deprecated）。
//
// URL 支持 HTTP(S) 链接或 data URI（"data:image/png;base64,..."）。
// 模型是否支持多模态取决于具体 Provider（GPT-4o / Claude 3+ / Gemini 等）。

// UserImageMessage 构造一条「图片 + 文本」用户消息。text 为空时仅含图片。
func UserImageMessage(imageURL, text string) *schema.Message {
	parts := []schema.MessageInputPart{imageInputPart(imageURL)}
	parts = appendTextPart(parts, text)
	return userMultiContentMessage(parts)
}

// UserImageMessages 构造一条「多图 + 文本」用户消息，图片顺序保留。
func UserImageMessages(imageURLs []string, text string) *schema.Message {
	parts := make([]schema.MessageInputPart, 0, len(imageURLs)+1)
	for _, u := range imageURLs {
		parts = append(parts, imageInputPart(u))
	}
	parts = appendTextPart(parts, text)
	return userMultiContentMessage(parts)
}

// UserAudioMessage 构造一条「音频 + 文本」用户消息（如 Gemini / GPT-4o-audio）。
func UserAudioMessage(audioURL, text string) *schema.Message {
	parts := []schema.MessageInputPart{{
		Type:  schema.ChatMessagePartTypeAudioURL,
		Audio: &schema.MessageInputAudio{MessagePartCommon: urlCommon(audioURL)},
	}}
	parts = appendTextPart(parts, text)
	return userMultiContentMessage(parts)
}

func imageInputPart(url string) schema.MessageInputPart {
	return schema.MessageInputPart{
		Type:  schema.ChatMessagePartTypeImageURL,
		Image: &schema.MessageInputImage{MessagePartCommon: urlCommon(url)},
	}
}

func appendTextPart(parts []schema.MessageInputPart, text string) []schema.MessageInputPart {
	if text == "" {
		return parts
	}
	return append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
}

func urlCommon(url string) schema.MessagePartCommon {
	return schema.MessagePartCommon{URL: &url}
}

func userMultiContentMessage(parts []schema.MessageInputPart) *schema.Message {
	return &schema.Message{Role: schema.User, UserInputMultiContent: parts}
}
