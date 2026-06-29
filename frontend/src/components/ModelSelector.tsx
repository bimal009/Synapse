import React from "react";
import { Cpu, Check, Plus, Settings2 } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ModelConfig } from "../types/types";

interface ModelSelectorProps {
  models: ModelConfig[];
  activeModelIds: string[];
  onToggle: (model: ModelConfig) => void;
  onManage: () => void;
  disabled?: boolean;
}

export function ModelSelector({
  models,
  activeModelIds,
  onToggle,
  onManage,
  disabled,
}: ModelSelectorProps) {
  const [open, setOpen] = React.useState(false);
  const isActive = (m: ModelConfig) => !!m.id && activeModelIds.includes(m.id);
  const activeCount = models.filter(isActive).length;

  return (
    <div className="flex items-center gap-1.5">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            disabled={disabled}
            className="h-7 gap-1.5 rounded-full px-2.5 text-xs"
          >
            <Cpu className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">
              {activeCount > 0 ? `${activeCount} active` : "Select models"}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-72 p-1.5">
          <div className="px-2 py-1.5">
            <p className="text-xs font-medium">Models</p>
            <p className="text-[11px] text-muted-foreground">
              Selected models run on every message.
            </p>
          </div>

          <div className="max-h-64 overflow-y-auto py-1">
            {models.length === 0 ? (
              <p className="px-2 py-6 text-center text-xs text-muted-foreground">
                No models yet.
              </p>
            ) : (
              models.map((m) => {
                const active = isActive(m);
                return (
                  <button
                    key={m.id || m.name}
                    type="button"
                    onClick={() => onToggle(m)}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors",
                      "hover:bg-accent",
                    )}
                  >
                    <span
                      className={cn(
                        "flex h-4 w-4 items-center justify-center rounded-[4px] border shrink-0",
                        active
                          ? "border-primary bg-primary text-primary-foreground"
                          : "border-border",
                      )}
                    >
                      {active && <Check className="h-3 w-3" />}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs font-medium">
                        {m.name || m.model || "Untitled model"}
                      </span>
                      <span className="block truncate text-[11px] text-muted-foreground">
                        {m.role ? `${m.role} · ` : ""}
                        {m.model}
                      </span>
                    </span>
                  </button>
                );
              })
            )}
          </div>

          <div className="border-t border-border pt-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-8 w-full justify-start gap-2 text-xs"
              onClick={() => {
                setOpen(false);
                onManage();
              }}
            >
              {models.length === 0 ? (
                <Plus className="h-3.5 w-3.5" />
              ) : (
                <Settings2 className="h-3.5 w-3.5" />
              )}
              {models.length === 0 ? "Add a model" : "Manage models"}
            </Button>
          </div>
        </PopoverContent>
      </Popover>

      {/* Active model chips */}
      {models.filter(isActive).map((m) => (
        <span
          key={m.id}
          className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary"
        >
          <Cpu className="h-3 w-3" />
          {m.role || m.name || m.model}
        </span>
      ))}
    </div>
  );
}
