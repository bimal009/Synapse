import React from "react";
import { Bot, Cpu, Plus, MessageSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ChatMessage, ChatSession } from "../types/types";

interface ChatAreaProps {
  hasSession: boolean;
  sessions: ChatSession[];
  messages: ChatMessage[];
  isLoading: boolean;
  liveMessage?: string;
  modelName: string;
  hasActiveModel: boolean;
  onOpenSettings: () => void;
  onNewSession: () => void;
}

export const ChatArea: React.FC<ChatAreaProps> = ({
  hasSession,
  sessions,
  messages,
  isLoading,
  liveMessage,
  modelName,
  hasActiveModel,
  onOpenSettings,
  onNewSession,
}) => {
  const bottomRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, isLoading, liveMessage]);

  // ── empty state: no chats at all ──────────────────────────────────────────
  if (!hasSession && sessions.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-6 p-8 select-none">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center">
            <Bot className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h1 className="text-lg font-semibold tracking-tight">Synapse</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              Local-first multi-agent AI coding assistant
            </p>
          </div>
        </div>

        {/* New chat CTA */}
        <Button size="sm" className="gap-2" onClick={onNewSession}>
          <Plus className="h-4 w-4" />
          Start your first chat
        </Button>

        {/* Model status */}
        {hasActiveModel ? (
          <StatusChip
            icon={<Cpu className="h-3 w-3" />}
            label={modelName}
            active
          />
        ) : (
          <div className="flex flex-col items-center gap-2">
            <p className="text-xs text-muted-foreground">
              No active model. Add or activate one to get started.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="h-7 gap-1.5 text-xs"
              onClick={onOpenSettings}
            >
              <Plus className="h-3 w-3" />
              Add a model
            </Button>
          </div>
        )}

        <p className="text-xs text-muted-foreground/60 max-w-[280px] text-center leading-relaxed">
          Select a project folder and start a conversation. Synapse agents can
          read and write files in your trusted workspace.
        </p>
      </div>
    );
  }

  // ── empty state: chats exist but none selected ────────────────────────────
  if (!hasSession) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-4 p-8 select-none">
        <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center">
          <MessageSquare className="h-5 w-5 text-muted-foreground" />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium">No chat selected</p>
          <p className="text-xs text-muted-foreground mt-0.5">
            Pick one from the sidebar or start a new one
          </p>
        </div>
        <Button size="sm" className="gap-2" onClick={onNewSession}>
          <Plus className="h-4 w-4" />
          New chat
        </Button>
      </div>
    );
  }

  // ── active chat ───────────────────────────────────────────────────────────
  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-3xl mx-auto px-4 py-6 space-y-4">
        {messages.length === 0 && !isLoading ? (
          <p className="text-center text-xs text-muted-foreground py-8">
            Send a message to start the conversation.
          </p>
        ) : (
          messages.map((m) => <MessageBubble key={m.id} message={m} />)
        )}
        {isLoading && (
          <div className="flex justify-start">
            <div className="rounded-2xl px-4 py-2.5 text-sm bg-muted text-muted-foreground animate-pulse">
              {liveMessage && liveMessage.trim() ? liveMessage : "Thinking…"}
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
};

// Lightweight inline markdown: **bold**, *italic*/_italic_, and `code`.
function renderInlineMarkdown(text: string): React.ReactNode[] {
  const pattern = /\*\*([^*]+)\*\*|`([^`]+)`|\*([^*]+)\*|_([^_]+)_/g;
  const nodes: React.ReactNode[] = [];
  let last = 0;
  let key = 0;
  let m: RegExpExecArray | null;

  while ((m = pattern.exec(text)) !== null) {
    if (m.index > last) nodes.push(text.slice(last, m.index));
    if (m[1] !== undefined) {
      nodes.push(<strong key={key++}>{m[1]}</strong>);
    } else if (m[2] !== undefined) {
      nodes.push(
        <code
          key={key++}
          className="rounded bg-foreground/10 px-1 py-0.5 font-mono text-[0.85em]"
        >
          {m[2]}
        </code>,
      );
    } else if (m[3] !== undefined) {
      nodes.push(<em key={key++}>{m[3]}</em>);
    } else if (m[4] !== undefined) {
      nodes.push(<em key={key++}>{m[4]}</em>);
    }
    last = pattern.lastIndex;
  }
  if (last < text.length) nodes.push(text.slice(last));
  return nodes;
}

const MessageBubble: React.FC<{ message: ChatMessage }> = ({ message }) => {
  const isUser = message.role === "user";
  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "rounded-2xl px-4 py-2.5 text-sm max-w-[85%] whitespace-pre-wrap break-words",
          isUser
            ? "bg-primary text-primary-foreground"
            : message.error
              ? "bg-destructive/10 text-destructive border border-destructive/20"
              : "bg-muted text-foreground",
        )}
      >
        {isUser ? message.content : renderInlineMarkdown(message.content)}
      </div>
    </div>
  );
};

const StatusChip: React.FC<{
  icon: React.ReactNode;
  label: string;
  active?: boolean;
}> = ({ icon, label, active }) => (
  <div
    className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-[11px] font-medium
    ${active ? "bg-primary/8 border-primary/25 text-primary" : "bg-muted/50 border-border text-muted-foreground"}`}
  >
    {icon}
    {label}
  </div>
);
