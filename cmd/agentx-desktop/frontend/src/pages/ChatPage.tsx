import { useState, useRef, useEffect, useCallback } from "react";
import type { ChatMessage } from "../lib/types";

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
}

let messageIdCounter = 0;

export default function ChatPage({ showToast }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
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
    <div className="flex flex-col h-full">
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
            <div className="text-center space-y-3">
              <div className="text-4xl opacity-20">💬</div>
              <p className="text-white/20 text-sm uppercase tracking-widest">
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
                <div className="text-sm whitespace-pre-wrap break-words leading-relaxed">
                  {streamingText}
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

function MessageBubble({ msg }: { msg: ChatMessage }) {
  return (
    <div
      className={`flex ${
        msg.role === "user" ? "justify-end" : "justify-start"
      }`}
    >
      <div
        className={`max-w-[80%] rounded-xl px-4 py-3 ${
          msg.role === "user"
            ? "bg-neon-pink/15 border border-neon-pink/30 text-white"
            : "bg-white/[0.04] border border-white/10 text-white/80"
        }`}
      >
        <div className="text-sm whitespace-pre-wrap break-words leading-relaxed">
          {msg.content}
        </div>
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
}
