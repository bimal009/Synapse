import { useEffect, useState, useCallback } from "react";
import { PromptInputBox } from "./components/ChatInput";
import { Sidebar } from "./components/Sidebar";
import { TopBar } from "./components/TopBar";
import { ChatArea } from "./components/ChatArea";
import { SettingsDialog } from "./components/SettingsDialog";
import { ModelSelector } from "./components/ModelSelector";
import ConfirmDialog from "./components/ConfirmDialog";
import {
  PermissionDialog,
  type PermissionResponse,
} from "./components/PermissionDialog";
import type { ChatSession, ChatMessage, ModelConfig } from "./types/types";
import {
  CreateChat,
  ListChats,
  DeleteChat,
  SetActiveChat,
  SelectFolder,
  TrustFolder,
  IsTrusted,
  GetPath,
  ListModels,
  AddModel,
  UpdateModel,
  DeleteModel,
  SetActiveModel,
  DeactivateModel,
  ActiveModelIDs,
  RunAgents,
} from "../wailsjs/go/main/App";
import { models as wails } from "../wailsjs/go/models";
import { EventsOn, EventsEmit } from "../wailsjs/runtime/runtime";

const isWails = () => typeof window !== "undefined" && "go" in window;

// ── helpers ───────────────────────────────────────────────────────────────────
function mapChat(c: wails.Chat): ChatSession {
  return {
    id: c.id,
    title: c.title || "Untitled",
    createdAt: new Date(c.created_at),
    updatedAt: c.last_opened ? new Date(c.last_opened) : new Date(c.created_at),
    projectPath: c.project_path ?? "",
  };
}

function mapModel(m: wails.Model): ModelConfig {
  return {
    id: m.id,
    name: m.name,
    role: m.role,
    model: m.model,
    url: m.url,
    api_key: m.api_key ?? "",
  };
}

async function fetchChats(): Promise<ChatSession[]> {
  const rows = await ListChats();
  return (rows ?? []).map(mapChat);
}

async function fetchModels(): Promise<{
  models: ModelConfig[];
  activeIds: string[];
}> {
  const [modelRows, activeIds] = await Promise.all([
    ListModels(),
    ActiveModelIDs(),
  ]);
  return {
    models: (modelRows ?? []).map(mapModel),
    activeIds: activeIds ?? [],
  };
}

// ── App ───────────────────────────────────────────────────────────────────────
function App() {
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [isTrusted, setIsTrusted] = useState(false);
  const [showTrustDialog, setShowTrustDialog] = useState(false);

  const [isLoading, setIsLoading] = useState(false);
  const [liveMessage, setLiveMessage] = useState("");
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [messagesBySession, setMessagesBySession] = useState<
    Record<string, ChatMessage[]>
  >({});

  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [notice, setNotice] = useState<string | null>(
    "Synapse is running locally. Models are served via Ollama.",
  );

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [activeModelIds, setActiveModelIds] = useState<string[]>([]);

  // Permission prompt raised by the backend (terminal "ask" flow).
  const [permissionAction, setPermissionAction] = useState<string | null>(null);

  const activeModel =
    models.find((m) => m.id && activeModelIds.includes(m.id)) ?? models[0];
  const hasActiveModel = activeModelIds.length > 0;

  // ── helpers ────────────────────────────────────────────────────────────────
  const updateSessionMessages = useCallback(
    (sessionId: string, fn: (prev: ChatMessage[]) => ChatMessage[]) => {
      setMessagesBySession((prev) => ({
        ...prev,
        [sessionId]: fn(prev[sessionId] ?? []),
      }));
    },
    [],
  );

  const refreshModels = useCallback(async () => {
    if (!isWails()) return;
    try {
      const { models: m, activeIds } = await fetchModels();
      setModels(m);
      setActiveModelIds(activeIds);
    } catch (err: unknown) {
      console.error("failed to refresh models:", err);
    }
  }, []);

  const refreshChats = useCallback(async () => {
    if (!isWails()) return;
    try {
      setSessions(await fetchChats());
    } catch (err: unknown) {
      console.error("failed to refresh chats:", err);
    }
  }, []);

  // ── startup ────────────────────────────────────────────────────────────────
  useEffect(() => {
    if (!isWails()) return;
    let cancelled = false;
    (async () => {
      try {
        const [chats, { models: m, activeIds }] = await Promise.all([
          fetchChats(),
          fetchModels(),
        ]);
        if (cancelled) return;
        setSessions(chats);
        setModels(m);
        setActiveModelIds(activeIds);
      } catch (err: unknown) {
        console.error("startup failed:", err);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // ── permission prompts ─────────────────────────────────────────────────────
  useEffect(() => {
    if (!isWails()) return;
    return EventsOn("ask:permission", (data: { action?: string }) => {
      setPermissionAction(data?.action ?? "");
    });
  }, []);

  // ── live activity messages from tool calls ─────────────────────────────────
  useEffect(() => {
    if (!isWails()) return;
    return EventsOn("live-message", (msg: string) => {
      setLiveMessage(typeof msg === "string" ? msg : "");
    });
  }, []);

  const respondPermission = useCallback((response: PermissionResponse) => {
    if (isWails()) EventsEmit("ask:permission:response", response);
    setPermissionAction(null);
  }, []);

  // ── switch active chat ─────────────────────────────────────────────────────
  const switchActiveChat = useCallback(async (chatId: string) => {
    setActiveSessionId(chatId);
    if (!isWails()) return;
    try {
      await SetActiveChat(chatId);
      const [{ models: m, activeIds }, trusted, path] = await Promise.all([
        fetchModels(),
        IsTrusted(),
        GetPath(),
      ]);
      setModels(m);
      setActiveModelIds(activeIds);
      setIsTrusted(trusted as boolean);
      setSelectedFolder((path as string) || null);
    } catch (err: unknown) {
      console.error("failed to switch chat:", err);
    }
  }, []);

  // ── folder ─────────────────────────────────────────────────────────────────
  const handleSelectFolder = useCallback(async () => {
    if (!isWails()) return;
    if (!activeSessionId) {
      setNotice("Start or select a chat before attaching a project folder.");
      return;
    }
    try {
      const path = await SelectFolder();
      if (path) {
        setSelectedFolder(path as string);
        setIsTrusted(false);
        setShowTrustDialog(true);
      }
    } catch (err: unknown) {
      console.error("failed to select folder:", err);
    }
  }, [activeSessionId]);

  const handleTrust = async () => {
    if (isWails()) {
      try {
        await TrustFolder();
        const path = await GetPath();
        setSelectedFolder((path as string) || null);
        await refreshModels();
      } catch (err: unknown) {
        console.error("failed to trust folder:", err);
      }
    }
    setIsTrusted(true);
    setShowTrustDialog(false);
  };

  const handleCancelTrust = () => {
    setSelectedFolder(null);
    setShowTrustDialog(false);
  };

  // ── model activate / deactivate ────────────────────────────────────────────
  const handleActivateModel = useCallback(
    async (model: ModelConfig) => {
      if (!isWails()) return;
      if (!model.id || !model.role.trim()) {
        setNotice("Save the model with a role before activating it.");
        return;
      }
      try {
        await SetActiveModel(model.role, model.id);
        await refreshModels();
      } catch (err: unknown) {
        setNotice(`Failed to activate model: ${err}`);
      }
    },
    [refreshModels],
  );

  const handleDeactivateModel = useCallback(
    async (model: ModelConfig) => {
      if (!isWails() || !model.id) return;
      try {
        await DeactivateModel(model.id);
        await refreshModels();
      } catch (err: unknown) {
        setNotice(`Failed to deactivate model: ${err}`);
      }
    },
    [refreshModels],
  );

  // Toggle a model's active state for the current chat (selected = active).
  const handleToggleModel = useCallback(
    (model: ModelConfig) => {
      if (!activeSessionId) {
        setNotice("Open or start a chat before selecting models.");
        return;
      }
      if (model.id && activeModelIds.includes(model.id)) {
        void handleDeactivateModel(model);
      } else {
        void handleActivateModel(model);
      }
    },
    [
      activeSessionId,
      activeModelIds,
      handleActivateModel,
      handleDeactivateModel,
    ],
  );

  // ── sessions ───────────────────────────────────────────────────────────────
  const handleNewSession = useCallback(async () => {
    if (!isWails()) return;
    try {
      const result = await CreateChat("New chat");
      const id = result as string;
      console.log("created chat id:", id); // check this in devtools
      if (!id) {
        console.error("CreateChat returned empty id");
        return;
      }
      await refreshChats();
      await switchActiveChat(id);
    } catch (err: unknown) {
      console.error("failed to create chat:", err);
    }
  }, [refreshChats, switchActiveChat]);

  const handleDeleteSession = useCallback(
    async (id: string) => {
      if (isWails()) {
        try {
          await DeleteChat(id);
        } catch (err: unknown) {
          console.error("failed to delete chat:", err);
        }
      }
      setSessions((p) => p.filter((s) => s.id !== id));
      setMessagesBySession((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
      if (activeSessionId === id) {
        setActiveSessionId(null);
        setSelectedFolder(null);
        setIsTrusted(false);
      }
    },
    [activeSessionId],
  );

  // ── send ───────────────────────────────────────────────────────────────────
  const handleSend = useCallback(
    async (message: string) => {
      if (isLoading) return;

      let sid = activeSessionId;

      if (!sid) {
        if (!isWails()) return;
        try {
          const id = await CreateChat(message.slice(0, 40) || "New chat");
          sid = id as string;
          await refreshChats();
          await switchActiveChat(sid);
        } catch (err: unknown) {
          console.error("failed to auto-create chat:", err);
          return;
        }
      } else {
        setSessions((p) =>
          p.map((s) => (s.id === sid ? { ...s, updatedAt: new Date() } : s)),
        );
      }

      const sessionId = sid;

      // show user message immediately
      updateSessionMessages(sessionId, (prev) => [
        ...prev,
        { id: crypto.randomUUID(), role: "user", content: message },
      ]);

      if (!isWails()) return;

      // no project attached — prompt for folder but don't block the message
      if (!isTrusted) {
        updateSessionMessages(sessionId, (prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: "assistant",
            content: "Please attach a project folder before running agents.",
            error: true,
          },
        ]);
        handleSelectFolder();
        return;
      }

      setIsLoading(true);
      setLiveMessage("");
      try {
        const reply = await RunAgents(message);
        updateSessionMessages(sessionId, (prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: "assistant",
            content: reply as string,
          },
        ]);
      } catch (err: unknown) {
        updateSessionMessages(sessionId, (prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: "assistant",
            content: String(err),
            error: true,
          },
        ]);
      } finally {
        setIsLoading(false);
        setLiveMessage("");
      }
    },
    [
      isLoading,
      activeSessionId,
      isTrusted,
      updateSessionMessages,
      refreshChats,
      switchActiveChat,
      handleSelectFolder,
    ],
  );

  // ── save models ────────────────────────────────────────────────────────────
  const handleSaveModels = async (updated: ModelConfig[]) => {
    if (!isWails()) {
      setModels(updated);
      return;
    }

    const invalid = updated.find(
      (m) =>
        !m.name.trim() || !m.role.trim() || !m.model.trim() || !m.url.trim(),
    );
    if (invalid) {
      setNotice("Each model needs a name, role, model and provider URL.");
      return;
    }

    try {
      const nextIds = new Set(updated.map((m) => m.id).filter(Boolean));
      for (const m of models) {
        if (m.id && !nextIds.has(m.id)) await DeleteModel(m.id);
      }
      for (const m of updated) {
        if (m.id) {
          await UpdateModel(
            wails.Model.createFrom({
              id: m.id,
              name: m.name,
              role: m.role,
              model: m.model,
              url: m.url,
              api_key: m.api_key,
            }),
          );
        } else {
          await AddModel(
            wails.Model.createFrom({
              name: m.name,
              role: m.role,
              model: m.model,
              url: m.url,
              api_key: m.api_key,
            }),
          );
        }
      }
      await refreshModels();
    } catch (err: unknown) {
      console.error("failed to save models:", err);
      setNotice(`Failed to save models: ${err}`);
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground">
      <Sidebar
        collapsed={sidebarCollapsed}
        onToggleCollapse={() => setSidebarCollapsed((p) => !p)}
        sessions={sessions}
        activeSessionId={activeSessionId}
        onSelectSession={switchActiveChat}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
        onOpenSettings={() => setSettingsOpen(true)}
        selectedFolder={selectedFolder}
        isTrusted={isTrusted}
        onSelectFolder={handleSelectFolder}
      />

      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <TopBar
          selectedFolder={selectedFolder}
          isTrusted={isTrusted}
          activeModelName={activeModel?.model ?? ""}
          notice={notice}
          onDismissNotice={() => setNotice(null)}
        />

        <ChatArea
          hasSession={!!activeSessionId}
          sessions={sessions}
          messages={
            activeSessionId ? (messagesBySession[activeSessionId] ?? []) : []
          }
          isLoading={isLoading}
          liveMessage={liveMessage}
          modelName={hasActiveModel ? (activeModel?.model ?? "") : ""}
          hasActiveModel={hasActiveModel}
          onOpenSettings={() => setSettingsOpen(true)}
          onNewSession={handleNewSession}
        />

        <div className="shrink-0 px-4 pb-4 pt-2">
          <div className="max-w-3xl mx-auto space-y-2">
            <ModelSelector
              models={models}
              activeModelIds={activeModelIds}
              onToggle={handleToggleModel}
              onManage={() => setSettingsOpen(true)}
              disabled={!activeSessionId}
            />
            <PromptInputBox
              onSend={handleSend}
              isLoading={isLoading}
              onCanvasToggle={handleSelectFolder}
              isProjectLoaded={isTrusted && !!selectedFolder}
            />
          </div>
        </div>
      </div>

      <SettingsDialog
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        models={models}
        onSaveModels={handleSaveModels}
      />

      <ConfirmDialog
        open={showTrustDialog}
        title="Do you trust this folder?"
        description={`You are opening:\n${selectedFolder}\n\nSynapse agents will be able to read and write files in this folder. Only trust folders you own.`}
        confirmLabel="Yes, trust folder"
        cancelLabel="Cancel"
        onConfirm={handleTrust}
        onCancel={handleCancelTrust}
      />

      <PermissionDialog
        open={permissionAction !== null}
        action={permissionAction ?? ""}
        onRespond={respondPermission}
      />
    </div>
  );
}

export default App;
