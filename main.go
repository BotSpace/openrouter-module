// openrouter-module-go — Botmother tashqi modul: OpenRouter LLM.
//
// Node turi:
//   - openrouter.Chat — action: OpenRouter orqali LLM'ga so'rov yuboradi,
//     javobni llm_output state'iga yozadi.
//
// OpenRouter = OpenAI-mos /chat/completions API (https://openrouter.ai).
// Credential: Bearer API key (https://openrouter.ai/keys).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	botmodule "github.com/BotSpace/botmodule-go"
)

const (
	moduleID = "openrouter"
	apiURL   = "https://openrouter.ai/api/v1/chat/completions"
)

var httpClient = &http.Client{Timeout: 120 * time.Second}

func main() {
	m := botmodule.New(moduleID, "OpenRouter")
	m.Version = "0.1.0"
	m.Docs = docs

	// Modul o'z credential turini e'lon qiladi — foydalanuvchi shu turdan
	// OpenRouter API key kiritadi. Key {slug}.* namespace bilan (platforma talabi).
	m.AddCredentialType(botmodule.CredentialType{
		Key:   "openrouter.apikey",
		Label: "OpenRouter API",
		Icon:  "brain-circuit",
		Color: "#6467F2",
		Mode:  "apikey",
		Fields: []botmodule.CredentialField{
			{
				Name:        "api_key",
				Label:       "API Key",
				Type:        "password",
				Required:    true,
				Secret:      true,
				Placeholder: "sk-or-...",
			},
		},
	})

	m.AddNode(botmodule.Node{
		Type:        "openrouter.Chat",
		Title:       "OpenRouter Chat",
		Description: "OpenRouter orqali LLM'ga so'rov yuboradi va javobni qaytaradi",
		Category:    "ai",
		Icon:        "brain-circuit",
		Color:       "ai-violet",
		Width:       200,
		Content: []botmodule.Field{
			{
				Type:           "credential",
				Key:            "api_credential",
				Label:          "OpenRouter API key",
				CredentialType: "openrouter.apikey",
				HelpText:       "https://openrouter.ai/keys dan oling",
			},
			{
				Type:        "text",
				Key:         "model",
				Label:       "Model",
				Placeholder: "openai/gpt-4o-mini",
				HelpText:    "https://openrouter.ai/models — masalan anthropic/claude-3.5-sonnet",
			},
			{
				Type:        "textarea",
				Key:         "system",
				Label:       "System prompt",
				Placeholder: "Sen foydali yordamchisan.",
				HelpText:    "Ixtiyoriy — modelning rolini belgilaydi",
				Optional:    true,
			},
			{
				Type:        "textarea",
				Key:         "prompt",
				Label:       "Foydalanuvchi xabari",
				Placeholder: "{{message.text}}",
				HelpText:    "Modelga yuboriladigan matn",
			},
			{
				Type:        "number",
				Key:         "temperature",
				Label:       "Temperature",
				Placeholder: "0.7",
				HelpText:    "0–2 oralig'ida; bo'sh = model defaulti",
				Optional:    true,
			},
		},
		Defaults: map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "{{message.text}}",
		},
		ProducesState: []string{"llm_output", "llm_model", "llm_tokens", "llm_error"},
		Execute:       executeChat,
	})

	m.Serve(":8100")
}

func executeChat(c *botmodule.ExecuteCtx) botmodule.Result {
	cred, ok := c.Credential("api_credential")
	if !ok {
		return errResult("API credential tanlanmagan")
	}
	apiKey := cred.Data["api_key"]
	if apiKey == "" {
		apiKey = cred.Data["token"] // bearer mode
	}
	if apiKey == "" {
		return errResult("API key bo'sh")
	}

	model := c.String("model")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	prompt := c.String("prompt")
	if prompt == "" {
		return errResult("prompt bo'sh")
	}

	messages := []map[string]string{}
	if sys := c.String("system"); sys != "" {
		messages = append(messages, map[string]string{"role": "system", "content": sys})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	payload := map[string]any{"model": model, "messages": messages}
	if temp, hasTemp := c.Data["temperature"]; hasTemp && temp != "" && temp != nil {
		payload["temperature"] = c.Int("temperature") // ponytail: number field; float kerak bo'lsa SDK'ga float helper qo'shilsin
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return errResult("so'rov qurilmadi: " + err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// OpenRouter reýting/atribut uchun ixtiyoriy headerlar:
	req.Header.Set("HTTP-Referer", "https://botspace.uz")
	req.Header.Set("X-Title", "Botmother")

	resp, err := httpClient.Do(req)
	if err != nil {
		return errResult("OpenRouter so'rovi muvaffaqiyatsiz: " + err.Error())
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errResult(fmt.Sprintf("OpenRouter %d: %s", resp.StatusCode, truncate(string(raw), 300)))
	}

	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return errResult("javob parse bo'lmadi: " + err.Error())
	}
	if out.Error.Message != "" {
		return errResult("OpenRouter: " + out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return errResult("javob bo'sh (choices yo'q)")
	}

	return botmodule.Result{
		ContextUpdates: map[string]any{
			"llm_output": out.Choices[0].Message.Content,
			"llm_model":  out.Model,
			"llm_tokens": out.Usage.TotalTokens,
			"llm_error":  "",
		},
		ExitOutput: "success",
	}
}

func errResult(msg string) botmodule.Result {
	return botmodule.Result{
		ContextUpdates: map[string]any{
			"llm_output": "",
			"llm_error":  msg,
		},
		ExitOutput: "error",
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

const docs = `# OpenRouter

[OpenRouter](https://openrouter.ai) orqali 300+ LLM modeliga yagona API bilan
murojaat qiladi (OpenAI, Anthropic, Google, Meta va boshqalar).

## Node turi

### ` + "`openrouter.Chat`" + ` (action, AI)

Tanlangan modelga chat so'rovi yuboradi va javobni state'ga yozadi.

| Field | Tavsif |
|---|---|
| **api_credential** | OpenRouter API key (Bearer). https://openrouter.ai/keys |
| **model** | Model slug, masalan ` + "`openai/gpt-4o-mini`" + `, ` + "`anthropic/claude-3.5-sonnet`" + ` |
| **system** | System prompt (ixtiyoriy) |
| **prompt** | Foydalanuvchi xabari, masalan ` + "`{{message.text}}`" + ` |
| **temperature** | 0–2 (ixtiyoriy) |

**Chiqish state'lari:**

- ` + "`llm_output`" + ` — model javobi
- ` + "`llm_model`" + ` — ishlatilgan model
- ` + "`llm_tokens`" + ` — sarflangan token soni
- ` + "`llm_error`" + ` — xato matni (muvaffaqiyatda bo'sh)

**Chiqish edge'lari:** ` + "`success`" + ` / ` + "`error`" + `

## Misol flow

` + "```" + `
Xabar kelganda (trigger)
  → OpenRouter Chat (prompt: {{message.text}})
  → Matn yuborish ({{llm_output}})
` + "```" + `
`
