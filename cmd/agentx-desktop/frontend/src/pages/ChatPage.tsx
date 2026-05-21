import React, { useState, useRef, useEffect, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { ChatMessage } from "../lib/types";
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

  const sendMessage = async () => {
    const text = input.trim();
    if (!text || sending) return;

    const userMsg: ChatMessage = {
      id: `msg-${++messageIdCounter}`,
      role: "user",
      content: text,
      timestamp: Date.now(),
    };

    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setSending(true);
    setStreamingText("");

    try {
      const resp = await window.go.main.ChatService.SendMessage(text, "");
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
        <div className="flex gap-3">
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
            disabled={!input.trim() || sending || connected === false}
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
          <div className="text-sm whitespace-pre-wrap break-words leading-relaxed">
            {msg.content}
          </div>
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
