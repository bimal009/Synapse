import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Eye, EyeOff, Plus, Trash2, Cpu } from "lucide-react";
import type { ModelConfig } from "../types/types";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  models: ModelConfig[];
  onSaveModels: (models: ModelConfig[]) => void;
}

const PRESET_URLS = [
  { label: "Ollama (local)", value: "http://localhost:11434" },
  { label: "OpenAI", value: "https://api.openai.com/v1" },
  { label: "Anthropic", value: "https://api.anthropic.com/v1" },
  { label: "Groq", value: "https://api.groq.com/openai/v1" },
  { label: "Custom", value: "" },
];

const DEFAULT_MODEL: ModelConfig = {
  id: "",
  name: "",
  role: "",
  model: "",
  url: "http://localhost:11434",
  api_key: "",
};

const ModelCard: React.FC<{
  model: ModelConfig;
  index: number;
  onChange: (updated: ModelConfig) => void;
  onRemove: () => void;
}> = ({ model, index, onChange, onRemove }) => {
  const [showKey, setShowKey] = React.useState(false);
  const [expanded, setExpanded] = React.useState(index === 0);

  const getPresetLabel = (url: string) => {
    const preset = PRESET_URLS.find((p) => p.value === url);
    return preset?.label ?? "Custom";
  };

  return (
    <div className="rounded-lg border border-border bg-card transition-colors hover:border-border-strong">
      {/* Card header */}
      <div
        className="flex items-center gap-3 px-3 py-2.5 cursor-pointer select-none"
        onClick={() => setExpanded((p) => !p)}
      >
        <div className="h-8 w-8 rounded-md flex items-center justify-center shrink-0 bg-muted">
          <Cpu className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">
            {model.name || model.model || `Model ${index + 1}`}
          </p>
          <p className="text-[11px] text-muted-foreground truncate">
            {model.role ? `${model.role} · ` : ""}
            {getPresetLabel(model.url)}
          </p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6 text-muted-foreground hover:text-destructive"
            onClick={(e) => {
              e.stopPropagation();
              onRemove();
            }}
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      </div>

      {/* Expanded fields */}
      {expanded && (
        <div className="border-t border-border px-3 py-3 space-y-3">
          {/* Display name */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Name</Label>
            <Input
              value={model.name}
              onChange={(e) => onChange({ ...model, name: e.target.value })}
              placeholder="e.g. Local planner"
              className="h-8 text-xs"
            />
          </div>

          {/* Role — a single role only (no commas / spaces) */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Role</Label>
            <Input
              value={model.role}
              onChange={(e) =>
                onChange({
                  ...model,
                  role: e.target.value.replace(/[,\s]/g, ""),
                })
              }
              placeholder="e.g. planner"
              className="h-8 text-xs"
            />
          </div>

          {/* Model identifier */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Model</Label>
            <Input
              value={model.model}
              onChange={(e) => onChange({ ...model, model: e.target.value })}
              placeholder="e.g. llama3.2, gpt-4o, claude-3-5-sonnet"
              className="h-8 text-xs"
            />
          </div>

          {/* URL preset + custom */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">
              Provider URL
            </Label>
            <Select
              value={
                PRESET_URLS.find((p) => p.value === model.url)?.value ?? ""
              }
              onValueChange={(v) => {
                if (v !== "") onChange({ ...model, url: v });
              }}
            >
              <SelectTrigger className="h-8 text-xs">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                {PRESET_URLS.map((p) => (
                  <SelectItem
                    key={p.value || "custom"}
                    value={p.value || "custom"}
                    className="text-xs"
                  >
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              value={model.url}
              onChange={(e) => onChange({ ...model, url: e.target.value })}
              placeholder="http://localhost:11434/v1"
              className="h-8 text-xs font-mono"
            />
          </div>

          {/* API Key */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">
              API key{" "}
              <span className="text-muted-foreground/50">(optional)</span>
            </Label>
            <div className="relative">
              <Input
                value={model.api_key}
                onChange={(e) => onChange({ ...model, api_key: e.target.value })}
                placeholder="sk-..."
                type={showKey ? "text" : "password"}
                className="h-8 text-xs pr-8 font-mono"
              />
              <button
                type="button"
                onClick={() => setShowKey((p) => !p)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              >
                {showKey ? (
                  <EyeOff className="h-3.5 w-3.5" />
                ) : (
                  <Eye className="h-3.5 w-3.5" />
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export const SettingsDialog: React.FC<SettingsDialogProps> = ({
  open,
  onOpenChange,
  models,
  onSaveModels,
}) => {
  const [localModels, setLocalModels] = React.useState<ModelConfig[]>(models);

  // Re-sync local draft state from props each time the dialog opens. Adjusting
  // state during render (rather than in an effect) avoids the cascading extra
  // render that `setState` in an effect would cause.
  const [prevOpen, setPrevOpen] = React.useState(open);
  if (open !== prevOpen) {
    setPrevOpen(open);
    if (open) {
      setLocalModels(models);
    }
  }

  const handleSave = () => {
    onSaveModels(localModels);
    onOpenChange(false);
  };

  const handleAddModel = () => {
    setLocalModels((p) => [...p, { ...DEFAULT_MODEL }]);
  };

  const handleRemoveModel = (i: number) => {
    setLocalModels((p) => p.filter((_, idx) => idx !== i));
  };

  const handleUpdateModel = (i: number, updated: ModelConfig) => {
    setLocalModels((p) => p.map((m, idx) => (idx === i ? updated : m)));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[560px] p-0 gap-0 overflow-hidden">
        <DialogHeader className="px-5 pt-5 pb-4 border-b border-border shrink-0">
          <DialogTitle className="text-base font-semibold">
            Settings
          </DialogTitle>
        </DialogHeader>

        <div className="overflow-y-auto max-h-[480px]">
          <div className="px-5 py-4 space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">Model configurations</p>
                <p className="text-xs text-muted-foreground">
                  Configure LLM providers and models
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-7 gap-1.5 text-xs"
                onClick={handleAddModel}
              >
                <Plus className="h-3 w-3" />
                Add model
              </Button>
            </div>

            {localModels.length === 0 ? (
              <div className="py-8 text-center rounded-lg border border-dashed border-border">
                <Cpu className="h-6 w-6 text-muted-foreground/30 mx-auto mb-2" />
                <p className="text-xs text-muted-foreground">
                  No models configured
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-2 h-7 text-xs"
                  onClick={handleAddModel}
                >
                  Add your first model
                </Button>
              </div>
            ) : (
              <div className="space-y-2">
                {localModels.map((model, i) => (
                  <ModelCard
                    key={i}
                    model={model}
                    index={i}
                    onChange={(updated) => handleUpdateModel(i, updated)}
                    onRemove={() => handleRemoveModel(i)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="mx-0 mb-0 px-5 py-3 border-t border-border bg-muted/20 shrink-0">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-xs"
          >
            Cancel
          </Button>
          <Button size="sm" onClick={handleSave} className="text-xs">
            Save changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
