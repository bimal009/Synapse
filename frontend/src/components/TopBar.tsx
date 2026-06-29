import React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Monitor, FolderOpen, GitBranch, X, Info } from "lucide-react";

interface TopBarProps {
  selectedFolder: string | null;
  isTrusted: boolean;
  activeModelName: string;
  onDismissNotice?: () => void;
  notice?: string | null;
}

export const TopBar: React.FC<TopBarProps> = ({
  selectedFolder,
  isTrusted,
  activeModelName,
  onDismissNotice,
  notice,
}) => {
  const folderName = selectedFolder
    ? selectedFolder.split("/").pop() ||
      selectedFolder.split("\\").pop() ||
      selectedFolder
    : null;

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex flex-col shrink-0">
        <div className="flex items-center h-10 px-3 border-b border-border bg-card gap-2">
          <div className="flex items-center gap-1.5 flex-1 min-w-0">
            <Chip icon={<Monitor className="h-3 w-3" />} label="Local" />

            {folderName && (
              <>
                <span className="text-muted-foreground/30 text-xs">/</span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Chip
                      icon={<FolderOpen className="h-3 w-3" />}
                      label={folderName}
                      active={isTrusted}
                    />
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className="text-xs">{selectedFolder}</p>
                    {isTrusted && (
                      <p className="text-xs text-muted-foreground">Trusted</p>
                    )}
                  </TooltipContent>
                </Tooltip>
              </>
            )}

            {isTrusted && (
              <>
                <span className="text-muted-foreground/30 text-xs">/</span>
                <Chip icon={<GitBranch className="h-3 w-3" />} label="main" />
              </>
            )}
          </div>

          {activeModelName && (
            <Badge
              variant="secondary"
              className="text-[10px] h-5 px-2 font-mono shrink-0"
            >
              {activeModelName}
            </Badge>
          )}
        </div>

        {notice && (
          <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/50 border-b border-border">
            <Info className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
            <p className="text-xs text-muted-foreground flex-1">{notice}</p>
            {onDismissNotice && (
              <Button
                variant="ghost"
                size="icon"
                className="h-5 w-5 shrink-0"
                onClick={onDismissNotice}
              >
                <X className="h-3 w-3" />
              </Button>
            )}
          </div>
        )}
      </div>
    </TooltipProvider>
  );
};

const Chip: React.FC<{
  icon: React.ReactNode;
  label: string;
  active?: boolean;
}> = ({ icon, label, active }) => (
  <div
    className={cn(
      "inline-flex items-center gap-1 px-2 h-6 rounded border text-[11px] font-medium transition-colors",
      active
        ? "bg-primary/10 border-primary/30 text-primary"
        : "bg-muted/50 border-border text-muted-foreground",
    )}
  >
    {icon}
    <span className="max-w-[120px] truncate">{label}</span>
  </div>
);
