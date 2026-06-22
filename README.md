# openrouter

Botmother tashqi modul: [OpenRouter](https://openrouter.ai) orqali 300+ LLM
modeliga (OpenAI, Anthropic, Google, Meta...) yagona API bilan murojaat.

## Node

| Type | Tur | Tavsif |
|---|---|---|
| `openrouter.Chat` | action (ai) | LLM'ga chat so'rovi yuboradi, javobni `llm_output` state'iga yozadi |

### Fieldlar

- **api_credential** — OpenRouter API key (Bearer). https://openrouter.ai/keys
- **model** — model slug (`openai/gpt-4o-mini`, `anthropic/claude-3.5-sonnet`, ...)
- **system** — system prompt (ixtiyoriy)
- **prompt** — foydalanuvchi xabari (`{{message.text}}`)
- **temperature** — 0–2 (ixtiyoriy)

### Chiqish

`llm_output`, `llm_model`, `llm_tokens`, `llm_error` + edge `success`/`error`.

## Lokal sinash

```bash
go run .
curl -X POST localhost:8100/rpc -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"describe","id":1}'
```

Faqat stdlib + `botmodule-go` SDK. To'liq ma'lumotnoma: `SDK.md` (template repo).

## Litsenziya

MIT.
