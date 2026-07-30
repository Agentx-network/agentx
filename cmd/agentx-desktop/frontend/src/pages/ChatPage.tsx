import React, { useState, useRef, useEffect, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { ChatMessage, Attachment, AttachmentKind } from "../lib/types";
import agentHero from "../assets/agent-hero.gif";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  messages: ChatMessage[];
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
}

let messageIdCounter = 0;

export default function ChatPage({ showToast, messages, setMessages }: Props) {
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [connected, setConnected] = useState<boolean | null>(null);
  const [streamingText, setStreamingText] = useState("");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [picking, setPicking] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, streamingText, scrollToBottom]);

  // An inline image grows the layout AFTER it finishes decoding, which would
  // otherwise leave the view parked mid-image. Re-scroll to the true bottom
  // once each image reports loaded. (Only fires on first load — cached images
  // are painted synchronously and never emit this.)
  useEffect(() => {
    window.addEventListener("agentx:image-loaded", scrollToBottom);
    return () => window.removeEventListener("agentx:image-loaded", scrollToBottom);
  }, [scrollToBottom]);

  // Poll for proactively-delivered messages (cron reminders that fire while the
  // user isn't mid-request). The gateway queues them for the "desktop" channel;
  // here we drain them and render each as an assistant message so a plain
  // "remind me in 1 min…" shows up right in this chat.
  useEffect(() => {
    let active = true;
    const poll = async () => {
      try {
        const msgs: string[] = await window.go.main.ChatService.PollNotifications();
        if (active && msgs && msgs.length > 0) {
          setMessages((prev) => [
            ...prev,
            ...msgs.map((content) => ({
              id: `msg-${++messageIdCounter}`,
              role: "assistant" as const,
              content,
              timestamp: Date.now(),
            })),
          ]);
        }
      } catch {
        // Gateway not reachable / no notifications — ignore and retry next tick.
      }
    };
    const interval = setInterval(poll, 4000);
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [setMessages]);

  // Check gateway connectivity
  useEffect(() => {
    const check = async () => {
      try {
        const ok = await window.go.main.ChatService.IsGatewayReachable();
        setConnected(ok);
      } catch {
        setConnected(false);
      }
    };
    check();
    const interval = setInterval(check, 10000);
    return () => clearInterval(interval);
  }, []);

  // Listen for streaming events from backend
  useEffect(() => {
    window.runtime.EventsOn("chat:delta", (delta: string) => {
      setStreamingText((prev) => prev + delta);
    });

    window.runtime.EventsOn("chat:done", (_response: string) => {
      // chat:done means the full response arrived; finalize handled in sendMessage
    });

    return () => {
      window.runtime.EventsOff("chat:delta");
      window.runtime.EventsOff("chat:done");
    };
  }, []);

  const handleAttach = async () => {
    if (picking || sending) return;
    setPicking(true);
    try {
      const res = await window.go.main.ChatService.PickAttachments();
      if (res.rejected?.length) {
        showToast(
          res.rejected.map((r) => `${r.name}: ${r.reason}`).join("  •  "),
          "error"
        );
      }
      if (res.accepted?.length) {
        setAttachments((prev) =>
          [...prev, ...(res.accepted as Attachment[])].slice(0, 8)
        );
      }
    } catch (e: any) {
      showToast(`${e}`, "error");
    } finally {
      setPicking(false);
    }
  };

  const removeAttachment = (path: string) =>
    setAttachments((prev) => prev.filter((a) => a.path !== path));

  const sendMessage = async () => {
    const text = input.trim();
    const atts = attachments;
    if ((!text && atts.length === 0) || sending) return;

    const userMsg: ChatMessage = {
      id: `msg-${++messageIdCounter}`,
      role: "user",
      content: text,
      timestamp: Date.now(),
      attachments: atts.length ? atts : undefined,
    };

    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setAttachments([]);
    setSending(true);
    setStreamingText("");

    try {
      const resp = await window.go.main.ChatService.SendMessage(
        text,
        "",
        atts.map((a) => a.path)
      );
      // Finalize: use the full response (streaming may have partial)
      const finalContent = resp.response || "";
      const assistantMsg: ChatMessage = {
        id: `msg-${++messageIdCounter}`,
        role: "assistant",
        content: finalContent,
        timestamp: Date.now(),
      };
      setMessages((prev) => [...prev, assistantMsg]);
      setStreamingText("");
    } catch (e: any) {
      showToast(`${e}`, "error");
      // If we got streaming text before error, keep it
      const partial = streamingText;
      const errorMsg: ChatMessage = {
        id: `msg-${++messageIdCounter}`,
        role: "assistant",
        content: partial || `Error: ${e}`,
        timestamp: Date.now(),
      };
      setMessages((prev) => [...prev, errorMsg]);
      setStreamingText("");
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex flex-col h-full min-h-0 flex-1 bg-[#0a0a12]/80 -m-6 p-6 rounded-xl">
      {/* Header */}
      <div className="flex items-center justify-between pb-4 border-b border-neon-pink/15">
        <h2 className="text-2xl font-bold uppercase tracking-[0.2em] text-glow-pink">
          Chat
        </h2>
        <div className="flex items-center gap-2">
          <div
            className={`w-2 h-2 rounded-full ${
              connected
                ? "bg-neon-green shadow-[0_0_8px_rgba(0,255,65,0.6)]"
                : connected === false
                ? "bg-red-500 shadow-[0_0_8px_rgba(255,0,0,0.4)]"
                : "bg-white/20"
            }`}
          />
          <span className="text-[10px] uppercase tracking-widest text-white/40">
            {connected
              ? "Connected"
              : connected === false
              ? "Offline"
              : "Checking..."}
          </span>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto py-4 space-y-4 min-h-0">
        {messages.length === 0 && !sending && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center space-y-4">
              <img src={agentHero} alt="" className="w-24 h-24 mx-auto rounded-full border-2 border-neon-pink/20 shadow-[0_0_30px_rgba(255,0,128,0.15)]" />
              <p className="text-white/25 text-sm uppercase tracking-widest">
                Start a conversation
              </p>
              <p className="text-white/10 text-xs max-w-xs">
                Send a message to your AgentX agent. Make sure the gateway is
                running.
              </p>
            </div>
          </div>
        )}

        {messages.map((msg) => (
          <MessageBubble key={msg.id} msg={msg} />
        ))}

        {/* Streaming assistant message */}
        {sending && (
          <div className="flex justify-start">
            <div className="max-w-[80%] rounded-xl px-4 py-3 bg-white/[0.04] border border-white/10 text-white/80">
              {streamingText ? (
                <div className="text-sm break-words leading-relaxed chat-markdown">
                  <MarkdownContent content={streamingText} />
                  <span className="inline-block w-1.5 h-4 bg-neon-pink/60 ml-0.5 animate-pulse" />
                </div>
              ) : (
                <div className="flex items-center gap-2 text-sm text-white/40">
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-neon-pink animate-pulse" />
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-neon-pink animate-pulse [animation-delay:0.2s]" />
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-neon-pink animate-pulse [animation-delay:0.4s]" />
                </div>
              )}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="pt-4 border-t border-neon-pink/15">
        {/* Pending attachment chips */}
        {attachments.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-3">
            {attachments.map((a) => (
              <PendingChip key={a.path} att={a} onRemove={() => removeAttachment(a.path)} />
            ))}
          </div>
        )}
        <div className="flex gap-3">
          <button
            onClick={handleAttach}
            disabled={sending || picking || connected === false}
            title="Attach files (images, audio, documents)"
            className="px-3 bg-white/[0.04] border-2 border-neon-purple/20 rounded-xl text-white/50 hover:text-neon-pink hover:border-neon-pink/40 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <PaperclipIcon />
          </button>
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={
              connected === false
                ? "Gateway offline..."
                : "Type a message..."
            }
            disabled={sending || connected === false}
            rows={1}
            className="flex-1 bg-white/[0.04] border-2 border-neon-purple/20 rounded-xl px-4 py-3 text-sm text-white placeholder-white/25 focus:outline-none focus:border-neon-pink/50 focus:shadow-neon-pink transition-all resize-none disabled:opacity-40"
          />
          <button
            onClick={sendMessage}
            disabled={(!input.trim() && attachments.length === 0) || sending || connected === false}
            className="px-5 bg-neon-pink text-white font-bold uppercase tracking-wider text-xs rounded-xl border border-neon-pink/60 hover:shadow-neon-pink active:scale-[0.97] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          >
            {sending ? "..." : "Send"}
          </button>
        </div>
        <p className="text-[10px] text-white/15 mt-2 text-center uppercase tracking-widest">
          Shift+Enter for new line
        </p>
      </div>
    </div>
  );
}

// IMAGE_MARKER matches an "IMAGE:<path>" line emitted by the image_generate
// tool. Captured group is the file path.
const IMAGE_MARKER = /^IMAGE:(.+)$/gm;

// ChatImage loads a locally-generated image as a data URL (via the Go binding)
// and renders it inline with a Download button. If the inline preview can't be
// read or decoded, it degrades to a clear message + Download so the user can
// always get the file.
// imageCache holds resolved data URLs by path so an image is fetched ONCE, not
// re-read on every chat re-render (e.g. each keystroke). Re-fetching is what
// caused the inline image to flicker and the view to jump.
const imageCache = new Map<string, string>();

const ChatImage = React.memo(function ChatImage({ path }: { path: string }) {
  const [src, setSrc] = useState<string | null>(() => imageCache.get(path) ?? null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    // Cache hit → nothing to do (no flicker, no refetch).
    if (imageCache.has(path)) {
      setSrc(imageCache.get(path)!);
      return;
    }
    let active = true;
    window.go.main.ChatService.ReadImageDataURL(path)
      .then((url) => { imageCache.set(path, url); if (active) setSrc(url); })
      .catch(() => { if (active) setFailed(true); });
    return () => { active = false; };
  }, [path]);

  const download = () => {
    window.go.main.ChatService.SaveImageAs(path).catch(() => {});
  };

  const DownloadBtn = (
    <button
      onClick={download}
      className="mt-1 text-[11px] uppercase tracking-widest text-neon-cyan/80 hover:text-neon-cyan"
    >
      ⬇ Download
    </button>
  );

  if (failed) {
    return (
      <div className="my-2">
        <div className="text-xs text-white/40">Preview unavailable — the image was saved to disk.</div>
        <div className="text-[11px] text-white/30 font-mono break-all">{path}</div>
        {DownloadBtn}
      </div>
    );
  }
  if (!src) {
    return <div className="text-xs text-white/30 my-2">Loading image…</div>;
  }
  return (
    <div className="my-2">
      <img
        src={src}
        alt="Generated image"
        onError={() => setFailed(true)}
        // When the image finishes loading the layout grows; tell the chat to
        // re-scroll so the latest message stays in view instead of landing
        // mid-image.
        onLoad={() => window.dispatchEvent(new Event("agentx:image-loaded"))}
        className="max-w-full rounded-lg border border-white/10 shadow-[0_0_20px_rgba(255,0,128,0.1)]"
      />
      <div>{DownloadBtn}</div>
    </div>
  );
});

// --- Attachments ---

// ATTACH_MARKER mirrors attach.DisplayMarker (Go). A stored user message is
// "<user text>[<docs note>]<ATTACH_MARKER><kind>|<path>|<name> lines". The
// visible text is everything before the first metadata marker; the attachment
// lines are parsed for re-render after a reload.
const ATTACH_MARKER = "\n\n[[attachments]]\n";
const DOCS_NOTE_MARKER = "\n\n[The user attached ";

// splitStoredUserContent separates a persisted user message into its visible
// text and its attachments (parsed from the marker block, if present).
function splitStoredUserContent(content: string): { text: string; attachments: Attachment[] } {
  let cut = content.length;
  const ai = content.indexOf(ATTACH_MARKER);
  const di = content.indexOf(DOCS_NOTE_MARKER);
  if (ai >= 0) cut = Math.min(cut, ai);
  if (di >= 0) cut = Math.min(cut, di);
  const text = content.slice(0, cut);

  const attachments: Attachment[] = [];
  if (ai >= 0) {
    const block = content.slice(ai + ATTACH_MARKER.length);
    for (const line of block.split("\n")) {
      if (!line.trim()) continue;
      const bar1 = line.indexOf("|");
      const bar2 = line.indexOf("|", bar1 + 1);
      if (bar1 < 0 || bar2 < 0) continue;
      const kind = line.slice(0, bar1) as AttachmentKind;
      const path = line.slice(bar1 + 1, bar2);
      const name = line.slice(bar2 + 1);
      if (path) attachments.push({ kind, path, name: name || path });
    }
  }
  return { text, attachments };
}

function KindIcon({ kind }: { kind: AttachmentKind }) {
  if (kind === "audio") {
    return (
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
        <path d="M9 3 5 6H2v4h3l4 3V3Z" /><path d="M11.5 5.5a3 3 0 0 1 0 5M13 4a5 5 0 0 1 0 8" />
      </svg>
    );
  }
  if (kind === "video") {
    return (
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
        <rect x="1.5" y="3.5" width="9" height="9" rx="1.5" /><path d="M10.5 6.5 14.5 4v8l-4-2.5Z" />
      </svg>
    );
  }
  // doc
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
      <path d="M9 1.5H4a1 1 0 0 0-1 1v11a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V5.5L9 1.5Z" /><path d="M9 1.5v4h4" />
    </svg>
  );
}

function PaperclipIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" className="mx-auto">
      <path d="M13 6.5 7.5 12a2.5 2.5 0 0 1-3.5-3.5l6-6a1.7 1.7 0 0 1 2.4 2.4l-6 6a.8.8 0 0 1-1.2-1.2L10.5 6" />
    </svg>
  );
}

// AttachThumb renders a small image preview from a local path (reusing the same
// data-URL binding + cache as ChatImage).
const AttachThumb = React.memo(function AttachThumb({ path, name }: { path: string; name: string }) {
  const [src, setSrc] = useState<string | null>(() => imageCache.get(path) ?? null);
  useEffect(() => {
    if (imageCache.has(path)) { setSrc(imageCache.get(path)!); return; }
    let active = true;
    window.go.main.ChatService.ReadImageDataURL(path)
      .then((url) => { imageCache.set(path, url); if (active) setSrc(url); })
      .catch(() => {});
    return () => { active = false; };
  }, [path]);
  if (!src) {
    return <div className="w-14 h-14 rounded-lg bg-white/[0.04] border border-white/10 animate-pulse" title={name} />;
  }
  return <img src={src} alt={name} title={name} className="w-14 h-14 object-cover rounded-lg border border-white/10" />;
});

// AttachmentList renders sent attachments inside a message bubble.
function AttachmentList({ items }: { items: Attachment[] }) {
  if (!items.length) return null;
  return (
    <div className="flex flex-wrap gap-2 mb-2">
      {items.map((a) =>
        a.kind === "image" ? (
          <AttachThumb key={a.path} path={a.path} name={a.name} />
        ) : (
          <div key={a.path} className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-white/[0.05] border border-white/10 text-white/70 max-w-[200px]">
            <KindIcon kind={a.kind} />
            <span className="text-xs truncate">{a.name}</span>
          </div>
        )
      )}
    </div>
  );
}

// PendingChip is an attachment queued in the composer (with a remove button).
function PendingChip({ att, onRemove }: { att: Attachment; onRemove: () => void }) {
  return (
    <div className="flex items-center gap-1.5 pl-1 pr-2 py-1 rounded-lg bg-white/[0.05] border border-neon-purple/20 text-white/75">
      {att.kind === "image" ? (
        <AttachThumb path={att.path} name={att.name} />
      ) : (
        <span className="pl-1"><KindIcon kind={att.kind} /></span>
      )}
      <span className="text-xs truncate max-w-[140px]">{att.name}</span>
      <button
        onClick={onRemove}
        title="Remove"
        className="text-white/40 hover:text-neon-pink text-sm leading-none px-1"
      >
        ✕
      </button>
    </div>
  );
}

// isLocalImagePath reports whether a markdown image src points at a local file
// (absolute path or file:// URL) rather than a remote http(s) URL or data URI.
// Weak models often render the generated image as `![alt](/local/path)` instead
// of emitting the IMAGE: marker; those local paths can't load in the WebView, so
// we route them through ChatImage (which reads them via the Go binding).
function isLocalImagePath(src: string): boolean {
  if (!src) return false;
  if (src.startsWith("data:") || src.startsWith("http://") || src.startsWith("https://")) return false;
  return src.startsWith("/") || src.startsWith("file://") || src.includes("/workspace/images/");
}

// AssistantContent renders markdown text, replacing any IMAGE:<path> markers
// with the rendered image inline.
function AssistantContent({ content }: { content: string }) {
  if (!content.includes("IMAGE:")) {
    return <MarkdownContent content={content} />;
  }
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let key = 0;
  content.replace(IMAGE_MARKER, (match, path: string, offset: number) => {
    const before = content.slice(lastIndex, offset).replace(/\n+$/, "");
    if (before.trim()) {
      parts.push(<MarkdownContent key={`t-${key}`} content={before} />);
    }
    parts.push(<ChatImage key={`i-${key}`} path={path.trim()} />);
    key++;
    lastIndex = offset + match.length;
    return match;
  });
  const rest = content.slice(lastIndex).replace(/^\n+/, "");
  if (rest.trim()) {
    parts.push(<MarkdownContent key={`t-${key}`} content={rest} />);
  }
  return <>{parts}</>;
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
        strong: ({ children }) => <strong className="font-bold text-white">{children}</strong>,
        em: ({ children }) => <em className="italic">{children}</em>,
        h1: ({ children }) => <h1 className="text-lg font-bold text-white mb-2">{children}</h1>,
        h2: ({ children }) => <h2 className="text-base font-bold text-white mb-2">{children}</h2>,
        h3: ({ children }) => <h3 className="text-sm font-bold text-white mb-1">{children}</h3>,
        ul: ({ children }) => <ul className="list-disc list-inside mb-2 space-y-0.5">{children}</ul>,
        ol: ({ children }) => <ol className="list-decimal list-inside mb-2 space-y-0.5">{children}</ol>,
        li: ({ children }) => <li className="text-sm">{children}</li>,
        code: ({ className, children }) => {
          const isBlock = className?.includes("language-");
          if (isBlock) {
            return (
              <code className="block bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-xs font-mono overflow-x-auto my-2">
                {children}
              </code>
            );
          }
          return (
            <code className="bg-white/10 px-1.5 py-0.5 rounded text-xs font-mono text-neon-pink/80">
              {children}
            </code>
          );
        },
        pre: ({ children }) => <pre className="my-2">{children}</pre>,
        img: ({ src, alt }) => {
          // A locally-generated image rendered as markdown (e.g. ![cat](/path))
          // can't load directly in the WebView — route it through ChatImage,
          // which reads the file via the Go binding. Remote URLs render as-is.
          if (typeof src === "string" && isLocalImagePath(src)) {
            return <ChatImage path={src} />;
          }
          return <img src={src} alt={alt} className="max-w-full rounded-lg border border-white/10 my-2" />;
        },
        a: ({ href, children }) => (
          <a href={href} target="_blank" rel="noopener noreferrer" className="text-neon-pink underline hover:text-neon-pink/80">
            {children}
          </a>
        ),
        hr: () => <hr className="border-white/10 my-3" />,
        blockquote: ({ children }) => (
          <blockquote className="border-l-2 border-neon-pink/30 pl-3 my-2 text-white/60 italic">{children}</blockquote>
        ),
        table: ({ children }) => (
          <div className="overflow-x-auto my-2">
            <table className="w-full text-xs border-collapse">{children}</table>
          </div>
        ),
        thead: ({ children }) => <thead className="border-b border-white/20">{children}</thead>,
        th: ({ children }) => <th className="text-left px-2 py-1.5 font-bold text-white/80 text-xs">{children}</th>,
        td: ({ children }) => <td className="px-2 py-1.5 border-b border-white/5 text-xs">{children}</td>,
      }}
    >
      {content}
    </ReactMarkdown>
  );
}

// Memoized so typing in the input (which re-renders ChatPage on every
// keystroke) does not re-render every existing bubble — each msg object is a
// stable reference, so settled bubbles skip re-rendering entirely.
const MessageBubble = React.memo(function MessageBubble({ msg }: { msg: ChatMessage }) {
  // For user messages, prefer live attachments; otherwise parse the persisted
  // marker block (reload path) and strip metadata from the visible text.
  let userText = msg.content;
  let userAtts = msg.attachments ?? [];
  if (msg.role === "user") {
    const parsed = splitStoredUserContent(msg.content);
    userText = parsed.text;
    if (userAtts.length === 0) userAtts = parsed.attachments;
  }

  return (
    <div
      className={`flex items-end gap-2 ${
        msg.role === "user" ? "justify-end" : "justify-start"
      }`}
    >
      {msg.role === "assistant" && (
        <img src={agentHero} alt="" className="w-7 h-7 rounded-full border border-neon-pink/20 flex-shrink-0 mb-1" />
      )}
      <div
        className={`max-w-[75%] rounded-xl px-4 py-3 ${
          msg.role === "user"
            ? "bg-neon-pink/15 border border-neon-pink/30 text-white"
            : "bg-white/[0.04] border border-white/10 text-white/80"
        }`}
      >
        {msg.role === "assistant" ? (
          <div className="text-sm break-words leading-relaxed chat-markdown">
            <AssistantContent content={msg.content} />
          </div>
        ) : (
          <>
            <AttachmentList items={userAtts} />
            {userText.trim() && (
              <div className="text-sm whitespace-pre-wrap break-words leading-relaxed">
                {userText}
              </div>
            )}
          </>
        )}
        <div
          className={`text-[10px] mt-1.5 ${
            msg.role === "user" ? "text-neon-pink/40" : "text-white/20"
          }`}
        >
          {new Date(msg.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
});
