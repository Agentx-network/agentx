# AgentX — KT Video Recording Script

**Target length:** 7–9 minutes · **Format:** screen recording + voice-over
**Goal:** the next engineer watches this once and understands *what the project is, what it's built with, how it runs, and where to start.*

> **How to use this script:** the left column is what you **do on screen**; the quoted text is what you **say**. Timings are a guide, not a rule. Speak naturally — you can paraphrase. Keep the codebase, the desktop app, and a terminal open in advance (see "Before you record" at the bottom).

---

## Scene 0 — Intro (0:00–0:40)

**On screen:** Your face-cam or just the README open in the editor (`README.md`).

> "Hi — I'm handing over the **AgentX** project. In this video I'll walk through what it is, the tech stack, how the code is organized, how it actually runs on a machine, and where to start reading. There's also a written KT doc — `docs/KT/AgentX-KT-Documentation.html` — that has everything in detail; this video is the guided tour.
>
> Quick framing: this repo is the **Go agent runtime and the desktop app** — internally we call it *OpenClaw*. It is **not** the website backend; that's a separate Python repo. I'll come back to that because it caused some confusion."

---

## Scene 1 — What the project is (0:40–1:40)

**On screen:** Scroll the top of `README.md` — the "About" and "What's New" sections.

> "AgentX is a **lightweight, self-hosted AI agent**. Think of a personal assistant that runs on your own machine: it chats, it can run shell commands, read and write files, search the web, generate images, and it learns 'skills' it can install on demand. It also has a built-in **crypto wallet** on BNB Smart Chain, and it can talk over messaging channels like Telegram.
>
> The whole thing is written in **Go** — it's tiny, a few megabytes of RAM, starts in under a second. That's the selling point versus other agent frameworks.
>
> It's the client/runtime half of a bigger product called the AgentX Network — the decentralized-agent-economy stuff, LaunchPad, marketplace — but *those* live in other repos. This repo is the engine and the desktop GUI."

---

## Scene 2 — Tech stack (1:40–2:40)

**On screen:** Open `go.mod`, then the `docs/KT` doc §2 (Technology stack table).

> "Tech stack, top to bottom:
> - **Go 1.26** is the core language — single static binary.
> - For talking to LLMs we use the **Fantasy SDK** — `charm.land/fantasy`. That's the abstraction over all the providers: OpenRouter, Anthropic, OpenAI, Gemini, Groq, and so on. It handles streaming, tool calls, and multimodal input.
> - The desktop app is **Wails v2** — that's Go on the backend with a web frontend in a native window. The frontend is **React, TypeScript, Tailwind, and Vite**.
> - The CLI uses **Cobra**.
> - It runs as a background service via **systemd** on Linux, launchd on Mac, Task Scheduler on Windows.
> - The wallet is **BNB Smart Chain**, and skills come from a marketplace called **ClawHub**."

---

## Scene 3 — Architecture: the three runtimes (2:40–4:00)

**On screen:** Open the KT doc §3 (the ASCII architecture diagram) and §4 (three-runtimes table).

> "Here's the mental model, and this is the single most important thing to understand.
>
> The same `agentx` binary runs in **three modes**:
> 1. As a **CLI**, for one-off commands.
> 2. As a **gateway** — a long-running background service. This is the actual brain. It listens on `localhost:18790`, runs the agent loop, connects the channels, runs scheduled tasks.
> 3. And the **desktop app**, which is a *separate* Wails binary.
>
> Now — the key insight: **the desktop app does not run the agent.** When you type in the chat window, it sends an HTTP request to the gateway — `POST /api/chat` — and streams the reply back over Server-Sent-Events. The gateway is where everything actually happens.
>
> So the desktop is a thin client, and the gateway is the backend. Remember this — it's why, if you change backend code, rebuilding just the desktop app does nothing. You have to rebuild the binary and restart the gateway. I'll show that."

**On screen:** Briefly show the request path — open `cmd/agentx/internal/gateway/helpers.go`, scroll to the `/api/chat` handler.

> "This is the `/api/chat` handler in the gateway. It takes the message, and calls into `pkg/agent` — the agent loop — which is the core."

---

## Scene 4 — Code tour (4:00–5:30)

**On screen:** Open the KT doc §5 (repo structure), then the file tree in the editor.

> "Quick tour of the layout. Two top-level places:
>
> **`cmd/`** has the two entry points — `cmd/agentx` is the CLI plus gateway, and `cmd/agentx-desktop` is the Wails app. In the desktop folder, the Go files like `chat.go` are the services the UI calls, and `frontend/src` is the React app — `ChatPage.tsx` is the chat screen, `ConfigPage.tsx` is settings.
>
> **`pkg/`** is the engine. The one to know is **`pkg/agent`** — that's the agent loop. `loop.go` and `fantasy_runner.go` are where a message becomes an LLM call, tools get executed, and a reply comes back."

**On screen:** Open `pkg/agent/loop.go`, scroll to `runAgentLoop`. Then open `pkg/tools/` folder.

> "The flow is: build the context — system prompt, identity, installed skills, memory, history — send it to the model, if the model asks to use a **tool** we run it and loop, otherwise we return the answer.
>
> Tools live in `pkg/tools` — `exec` for shell commands, file read/write, web search, image generation, skill install, cron for reminders. `exec` is the powerful one — it's how the agent actually *does* things, like running ffmpeg to compress a video.
>
> Other packages worth knowing: `pkg/session` is chat memory, `pkg/skills` is the ClawHub client, `pkg/channels` is Telegram/Discord/etc., `pkg/wallet` is the BSC wallet, and `pkg/providers` bridges to the Fantasy SDK."

---

## Scene 5 — Memory (important nuance) (5:30–6:15)

**On screen:** Open `~/.agentx/workspace/sessions/` in a file manager or terminal `ls`. Show a `.json` and a `.transcript.jsonl`.

> "One nuance on memory, because it surprised us. Each conversation is stored on disk in `~/.agentx/workspace/sessions/`.
>
> There are **two files per session**. The `.json` is what the *model* sees — and when a chat gets long, older messages get **summarized and deleted** to stay within the token budget. That's by design.
>
> But that meant the chat window lost old messages after a restart. So there's now a second file, the `.transcript.jsonl` — an append-only log that's **never** trimmed. The desktop reads that for display. So: the agent's memory is compacted for efficiency, but the UI keeps the full history. Two separate concerns, two files."

---

## Scene 6 — Live demo: how it runs (6:15–7:45)

**On screen:** A terminal.

> "Let me show it running. The gateway is a systemd user service:"

**Do:** run
```
systemctl --user status agentx-gateway.service
ss -tlnp | grep 18790
```
> "There it is — listening on 18790. All the data is under `~/.agentx`:"

**Do:** run `ls ~/.agentx` and `ls ~/.agentx/workspace`
> "`config.json` is the settings — providers, model, keys. The `workspace` has the identity files, the sessions, installed skills, and uploads."

**On screen:** Switch to the **desktop app** (already open). Show the **Dashboard**.

> "This is the desktop app. The Dashboard shows the agent's status, the model — right now it's set to 'Auto', which routes through OpenRouter — the enabled channels, the wallet balances, and live logs."

**On screen:** Go to **Chat**. Send a message; show it stream. Click the **paperclip**, attach a file (e.g. a PDF or an image), send, show the answer.

> "Here's chat — replies stream in live. And this paperclip is file upload: you can attach a document and it reads it, an image and it looks at it, or a video and it'll process it with ffmpeg. That's some of the work from this last stretch."

**On screen:** Show the redeploy loop in the terminal (don't run fully, just narrate):
```
make build
systemctl --user stop agentx-gateway.service
cp build/agentx-linux-amd64 ~/.local/bin/agentx
systemctl --user start agentx-gateway.service
make desktop-build
```
> "And this is the redeploy loop: rebuild the binary, swap it in, restart the gateway — because, again, chat runs in the gateway, not the app. The KT doc has this exact sequence."

---

## Scene 7 — Gotchas & where to start (7:45–8:45)

**On screen:** KT doc §13 (gotchas table).

> "A few gotchas the doc covers in detail — I'll flag the big ones:
> - **Backend changes need a gateway restart.** Rebuilding the app alone won't do it.
> - The desktop can **repoint the gateway service to the old binary** on `/usr/local/bin`, so the clean deploy is to install your new build there with sudo.
> - The **OpenRouter key was low on credits** — if chat fails with a 402, that's why; fund it or set a specific model.
> - And there was a whole **Infura** mix-up: Infura is **not in this repo** — it's in the separate `agentx-backend` Python/MongoDB project. Don't go looking for it here.
>
> Where to start reading: `pkg/agent/loop.go`, then `fantasy_runner.go`, then the gateway's `helpers.go`. That's the entire request path. The KT doc has a step-by-step onboarding checklist at the end."

---

## Scene 8 — Outro (8:45–9:00)

**On screen:** The KT doc title page, or the `docs/KT/` folder.

> "That's the tour. Everything here — architecture, deployment, the recent changes, the gotchas, a cheat sheet, and an onboarding checklist — is in `docs/KT/AgentX-KT-Documentation`. Read that alongside this video and you'll be productive quickly. Good luck — thanks!"

---

## Before you record — checklist

- [ ] **Windows/tabs open in advance:** the editor on this repo, the running desktop app, a terminal, and the KT doc (open the HTML in a browser).
- [ ] **Start the gateway** so the demo works: `systemctl --user start agentx-gateway.service` (and confirm the model/provider key has credit, or set a working one, so the live chat doesn't error).
- [ ] **Pre-stage a file** to attach in the chat demo (a small PDF or image).
- [ ] **Clean desktop** — close noisy apps/notifications; hide any secrets (API keys, wallet private key — do **not** show `wallet.json`, `config.json` values, or the export-key screen on camera).
- [ ] **Zoom the editor font** up (~16–18px) so code is legible in the recording.
- [ ] Do a 20-second test recording to check mic level and that the screen capture is sharp.

## Recording tools (pick one)

| OS | Free tool |
|----|-----------|
| Linux | **OBS Studio** (full control), or **GNOME Screencast** (<kbd>Ctrl+Alt+Shift+R</kbd>) for a quick capture |
| macOS | **QuickTime** (File → New Screen Recording), or OBS |
| Windows | **Xbox Game Bar** (<kbd>Win+G</kbd>), or OBS |
| Cross-platform, polished | **OBS Studio** — record 1080p, 30 fps, system audio off, mic on |

**Tips:** record in 1080p; keep mouse movements slow and deliberate; pause a beat before switching windows; if you fluff a line, pause and repeat the sentence — you can trim later. A light edit (trim dead air, maybe captions) in DaVinci Resolve (free) or Clipchamp/iMovie is plenty.
