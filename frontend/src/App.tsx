import { useEffect, useState } from "react";
import { PromptInputBox } from "./components/ChatInput";

import { GetProject, SelectFolder, TrustFolder } from "../wailsjs/go/main/App";
import ConfirmDialog from "./components/ConfirmDialog";

function App() {
  const [isLoading, setIsLoading] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [isTrusted, setIsTrusted] = useState(false);
  const [showTrustDialog, setShowTrustDialog] = useState(false);

  // Restore the last trusted project from the database (Go) on startup.
  useEffect(() => {
    GetProject()
      .then((path) => {
        if (path) {
          setSelectedFolder(path);
          setIsTrusted(true);
        }
      })
      .catch((err) => console.error("failed to load project:", err));
  }, []);

  const handleSelectFolder = async () => {
    const path = await SelectFolder();
    if (path) {
      setSelectedFolder(path);
      setIsTrusted(false);
      setShowTrustDialog(true);
    }
  };

  const handleTrust = async () => {
    await TrustFolder();
    setIsTrusted(true);
    setShowTrustDialog(false);
  };

  const handleCancelTrust = () => {
    setSelectedFolder(null);
    setShowTrustDialog(false);
  };

  const handleSend = (message: string, files?: File[]) => {
    if (!isTrusted) {
      handleSelectFolder();
      return;
    }
    console.log("Message:", message);
    console.log("Files:", files);
    setIsLoading(true);
    setTimeout(() => setIsLoading(false), 3000);
  };

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4 gap-4">
      <div className="w-full max-w-2xl">
        <PromptInputBox
          onSend={handleSend}
          isLoading={isLoading}
          onCanvasToggle={handleSelectFolder}
          isProjectLoaded={isTrusted && !!selectedFolder}
          placeholder={
            isTrusted
              ? "Ask me anything..."
              : "Select a project folder to get started..."
          }
        />
      </div>

      <ConfirmDialog
        open={showTrustDialog}
        title="Do you trust this folder?"
        description={`You are opening: ${selectedFolder}\n\nSynapse agents will be able to read and write files in this folder. Only trust folders you own.`}
        confirmLabel="Yes, trust folder"
        cancelLabel="Cancel"
        onConfirm={handleTrust}
        onCancel={handleCancelTrust}
      />
    </div>
  );
}

export default App;
