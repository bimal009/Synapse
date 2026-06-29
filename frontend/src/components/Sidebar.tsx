import React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  MessageSquare,
  Plus,
  Settings,
  FolderOpen,
  ChevronLeft,
  ChevronRight,
  Trash2,
  MoreHorizontal,
  Bot,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { ChatSession } from "@/types/types";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  sessions: ChatSession[];
  activeSessionId: string | null;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
  onDeleteSession: (id: string) => void;
  onOpenSettings: () => void;
  selectedFolder: string | null;
  isTrusted: boolean;
  onSelectFolder: () => void;
}

function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return `${days}d ago`;
}

function groupSessions(sessions: ChatSession[]) {
  const now = new Date();
  const today: ChatSession[] = [];
  const yesterday: ChatSession[] = [];
  const older: ChatSession[] = [];

  sessions.forEach((s) => {
    const diff = now.getTime() - s.updatedAt.getTime();
    const days = diff / 86400000;
    if (days < 1) today.push(s);
    else if (days < 2) yesterday.push(s);
    else older.push(s);
  });

  return { today, yesterday, older };
}

const SessionItem: React.FC<{
  session: ChatSession;
  isActive: boolean;
  collapsed: boolean;
  onSelect: () => void;
  onDelete: () => void;
}> = ({ session, isActive, collapsed, onSelect, onDelete }) => {
  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={onSelect}
            className={cn(
              "w-full flex items-center justify-center h-9 rounded-md transition-colors",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
            )}
          >
            <MessageSquare className="h-4 w-4 shrink-0" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="right">
          <p className="text-xs">{session.title}</p>
        </TooltipContent>
      </Tooltip>
    );
  }

  return (
    <div
      className={cn(
        "group relative flex items-center gap-2 px-2 py-1.5 rounded-md cursor-pointer transition-colors",
        isActive
          ? "bg-accent text-accent-foreground"
          : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
      )}
      onClick={onSelect}
    >
      <MessageSquare className="h-3.5 w-3.5 shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium truncate leading-none mb-0.5">
          {session.title}
        </p>
        <p className="text-[10px] text-muted-foreground/70">
          {formatRelativeTime(session.updatedAt)}
        </p>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            className="opacity-0 group-hover:opacity-100 h-5 w-5 flex items-center justify-center rounded hover:bg-background/50 transition-all"
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="h-3 w-3" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-32">
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
            className="text-destructive focus:text-destructive text-xs"
          >
            <Trash2 className="h-3 w-3 mr-2" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};

const SectionLabel: React.FC<{ label: string; collapsed: boolean }> = ({
  label,
  collapsed,
}) => {
  if (collapsed) return <div className="h-px bg-border mx-1 my-1" />;
  return (
    <p className="px-2 pt-3 pb-1 text-[10px] font-medium text-muted-foreground/60 uppercase tracking-wider">
      {label}
    </p>
  );
};

export const Sidebar: React.FC<SidebarProps> = ({
  collapsed,
  onToggleCollapse,
  sessions,
  activeSessionId,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  onOpenSettings,
  selectedFolder,
  isTrusted,
  onSelectFolder,
}) => {
  const { today, yesterday, older } = groupSessions(sessions);

  const folderName = selectedFolder
    ? selectedFolder.split("/").pop() ||
      selectedFolder.split("\\").pop() ||
      selectedFolder
    : null;

  return (
    <TooltipProvider delayDuration={300}>
      <aside
        className={cn(
          "flex flex-col h-full border-r border-border bg-card transition-all duration-300 ease-in-out shrink-0",
          collapsed ? "w-[52px]" : "w-[220px]",
        )}
      >
        {/* Header */}
        <div
          className={cn(
            "flex items-center h-11 px-2 border-b border-border shrink-0",
            collapsed ? "justify-center" : "justify-between",
          )}
        >
          {!collapsed && (
            <div className="flex items-center gap-1.5 min-w-0">
              <Bot className="h-4 w-4 text-primary shrink-0" />
              <span className="text-sm font-semibold tracking-tight truncate">
                Synapse
              </span>
            </div>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                onClick={onToggleCollapse}
              >
                {collapsed ? (
                  <ChevronRight className="h-3.5 w-3.5" />
                ) : (
                  <ChevronLeft className="h-3.5 w-3.5" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">
              {collapsed ? "Expand sidebar" : "Collapse sidebar"}
            </TooltipContent>
          </Tooltip>
        </div>

        {/* New Chat */}
        <div className={cn("px-2 pt-2 pb-1 shrink-0", collapsed && "px-1.5")}>
          {collapsed ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={onNewSession}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">New chat</TooltipContent>
            </Tooltip>
          ) : (
            <Button
              variant="outline"
              size="sm"
              className="w-full h-8 justify-start gap-2 text-xs font-medium"
              onClick={onNewSession}
            >
              <Plus className="h-3.5 w-3.5" />
              New chat
            </Button>
          )}
        </div>

        {/* Sessions */}
        <ScrollArea className="flex-1 px-2 py-1">
          <div className={cn("space-y-0.5", collapsed && "px-0")}>
            {today.length > 0 && (
              <>
                <SectionLabel label="Today" collapsed={collapsed} />
                {today.map((s) => (
                  <SessionItem
                    key={s.id}
                    session={s}
                    isActive={s.id === activeSessionId}
                    collapsed={collapsed}
                    onSelect={() => onSelectSession(s.id)}
                    onDelete={() => onDeleteSession(s.id)}
                  />
                ))}
              </>
            )}
            {yesterday.length > 0 && (
              <>
                <SectionLabel label="Yesterday" collapsed={collapsed} />
                {yesterday.map((s) => (
                  <SessionItem
                    key={s.id}
                    session={s}
                    isActive={s.id === activeSessionId}
                    collapsed={collapsed}
                    onSelect={() => onSelectSession(s.id)}
                    onDelete={() => onDeleteSession(s.id)}
                  />
                ))}
              </>
            )}
            {older.length > 0 && (
              <>
                <SectionLabel label="Earlier" collapsed={collapsed} />
                {older.map((s) => (
                  <SessionItem
                    key={s.id}
                    session={s}
                    isActive={s.id === activeSessionId}
                    collapsed={collapsed}
                    onSelect={() => onSelectSession(s.id)}
                    onDelete={() => onDeleteSession(s.id)}
                  />
                ))}
              </>
            )}
            {sessions.length === 0 && !collapsed && (
              <div className="px-2 py-8 text-center">
                <MessageSquare className="h-6 w-6 text-muted-foreground/30 mx-auto mb-2" />
                <p className="text-xs text-muted-foreground/50">No chats yet</p>
              </div>
            )}
          </div>
        </ScrollArea>

        <Separator />

        {/* Footer */}
        <div
          className={cn("px-2 py-2 space-y-1 shrink-0", collapsed && "px-1.5")}
        >
          {/* Project folder */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={onSelectFolder}
                className={cn(
                  "w-full flex items-center gap-2 px-2 py-1.5 rounded-md transition-colors text-left",
                  isTrusted && selectedFolder
                    ? "text-foreground hover:bg-accent/50"
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                )}
              >
                <FolderOpen
                  className={cn(
                    "h-4 w-4 shrink-0",
                    isTrusted && selectedFolder
                      ? "text-primary"
                      : "text-muted-foreground",
                  )}
                />
                {!collapsed && (
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium truncate leading-none">
                      {folderName || "Open project"}
                    </p>
                    {isTrusted && selectedFolder && (
                      <p className="text-[10px] text-muted-foreground/60 truncate">
                        Trusted
                      </p>
                    )}
                  </div>
                )}
              </button>
            </TooltipTrigger>
            {collapsed && (
              <TooltipContent side="right">
                {folderName ? `Project: ${folderName}` : "Open project"}
              </TooltipContent>
            )}
          </Tooltip>

          {/* Settings */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={onOpenSettings}
                className={cn(
                  "w-full flex items-center gap-2 px-2 py-1.5 rounded-md transition-colors text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                )}
              >
                <Settings className="h-4 w-4 shrink-0" />
                {!collapsed && (
                  <span className="text-xs font-medium">Settings</span>
                )}
              </button>
            </TooltipTrigger>
            {collapsed && (
              <TooltipContent side="right">Settings</TooltipContent>
            )}
          </Tooltip>
        </div>
      </aside>
    </TooltipProvider>
  );
};
