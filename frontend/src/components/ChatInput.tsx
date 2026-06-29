import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  ArrowUp,
  Paperclip,
  Square,
  X,
  StopCircle,
  Mic,
  Globe,
  FolderOpen,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

// ─── VoiceRecorder ───────────────────────────────────────────────────────────

interface VoiceRecorderProps {
  isRecording: boolean;
  onStartRecording: () => void;
  onStopRecording: (duration: number) => void;
  visualizerBars?: number;
}

const VoiceRecorder: React.FC<VoiceRecorderProps> = ({
  isRecording,
  onStartRecording,
  onStopRecording,
  visualizerBars = 32,
}) => {
  const [time, setTime] = React.useState(0);
  const timeRef = React.useRef(0);

  React.useEffect(() => {
    onStartRecording();
    const id = setInterval(() => {
      timeRef.current += 1;
      setTime(timeRef.current);
    }, 1000);
    return () => {
      clearInterval(id);
      onStopRecording(timeRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const bars = React.useMemo(
    () =>
      Array.from({ length: visualizerBars }, (_, i) => {
        const frac = (n: number) => {
          const x = Math.sin(n) * 43758.5453;
          return x - Math.floor(x);
        };
        return {
          height: Math.max(15, frac(i * 12.9898) * 100),
          duration: 0.5 + frac(i * 78.233) * 0.5,
        };
      }),
    [visualizerBars],
  );

  const formatTime = (s: number) =>
    `${Math.floor(s / 60)
      .toString()
      .padStart(2, "0")}:${(s % 60).toString().padStart(2, "0")}`;

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center w-full transition-all duration-300 py-3",
        isRecording ? "opacity-100" : "opacity-0 h-0",
      )}
    >
      <div className="flex items-center gap-2 mb-3">
        <div className="h-2 w-2 rounded-full bg-destructive animate-pulse" />
        <span className="font-mono text-sm text-muted-foreground">
          {formatTime(time)}
        </span>
      </div>
      <div className="w-full h-10 flex items-center justify-center gap-0.5 px-4">
        {bars.map((bar, i) => (
          <div
            key={i}
            className="w-0.5 rounded-full bg-primary/50 animate-pulse"
            style={{
              height: `${bar.height}%`,
              animationDelay: `${i * 0.05}s`,
              animationDuration: `${bar.duration}s`,
            }}
          />
        ))}
      </div>
    </div>
  );
};

// ─── ImageViewDialog ─────────────────────────────────────────────────────────

interface ImageViewDialogProps {
  imageUrl: string | null;
  onClose: () => void;
}

const ImageViewDialog: React.FC<ImageViewDialogProps> = ({
  imageUrl,
  onClose,
}) => {
  if (!imageUrl) return null;
  return (
    <Dialog open={!!imageUrl} onOpenChange={onClose}>
      <DialogContent className="p-0 border-none bg-transparent shadow-none max-w-[90vw] md:max-w-[800px]">
        <DialogTitle className="sr-only">Image preview</DialogTitle>
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          className="relative bg-card rounded-2xl overflow-hidden shadow-2xl"
        >
          <img
            src={imageUrl}
            alt="Full preview"
            className="w-full max-h-[80vh] object-contain rounded-2xl"
          />
        </motion.div>
      </DialogContent>
    </Dialog>
  );
};

// ─── PromptInput Context ──────────────────────────────────────────────────────

interface PromptInputContextType {
  isLoading: boolean;
  value: string;
  setValue: (value: string) => void;
  maxHeight: number | string;
  onSubmit?: () => void;
  disabled?: boolean;
}

const PromptInputContext = React.createContext<PromptInputContextType>({
  isLoading: false,
  value: "",
  setValue: () => {},
  maxHeight: 240,
  onSubmit: undefined,
  disabled: false,
});

function usePromptInput() {
  return React.useContext(PromptInputContext);
}

// ─── PromptInput ──────────────────────────────────────────────────────────────

interface PromptInputProps {
  isLoading?: boolean;
  value?: string;
  onValueChange?: (value: string) => void;
  maxHeight?: number | string;
  onSubmit?: () => void;
  children: React.ReactNode;
  className?: string;
  disabled?: boolean;
  onDragOver?: (e: React.DragEvent) => void;
  onDragLeave?: (e: React.DragEvent) => void;
  onDrop?: (e: React.DragEvent) => void;
}

const PromptInput = React.forwardRef<HTMLDivElement, PromptInputProps>(
  (
    {
      className,
      isLoading = false,
      maxHeight = 240,
      value,
      onValueChange,
      onSubmit,
      children,
      disabled = false,
      onDragOver,
      onDragLeave,
      onDrop,
    },
    ref,
  ) => {
    const [internalValue, setInternalValue] = React.useState(value || "");
    const handleChange = (newValue: string) => {
      setInternalValue(newValue);
      onValueChange?.(newValue);
    };
    return (
      <TooltipProvider>
        <PromptInputContext.Provider
          value={{
            isLoading,
            value: value ?? internalValue,
            setValue: onValueChange ?? handleChange,
            maxHeight,
            onSubmit,
            disabled,
          }}
        >
          <div
            ref={ref}
            className={cn(
              "rounded-2xl border border-border bg-card p-2 shadow-sm transition-all duration-300",
              isLoading && "border-primary/40",
              className,
            )}
            onDragOver={onDragOver}
            onDragLeave={onDragLeave}
            onDrop={onDrop}
          >
            {children}
          </div>
        </PromptInputContext.Provider>
      </TooltipProvider>
    );
  },
);
PromptInput.displayName = "PromptInput";

// ─── PromptInputTextarea ──────────────────────────────────────────────────────

interface PromptInputTextareaProps {
  disableAutosize?: boolean;
  placeholder?: string;
}

const PromptInputTextarea: React.FC<
  PromptInputTextareaProps & React.ComponentProps<typeof Textarea>
> = ({
  className,
  onKeyDown,
  disableAutosize = false,
  placeholder,
  ...props
}) => {
  const { value, setValue, maxHeight, onSubmit, disabled } = usePromptInput();
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);

  React.useEffect(() => {
    if (disableAutosize || !textareaRef.current) return;
    textareaRef.current.style.height = "auto";
    textareaRef.current.style.height =
      typeof maxHeight === "number"
        ? `${Math.min(textareaRef.current.scrollHeight, maxHeight)}px`
        : `min(${textareaRef.current.scrollHeight}px, ${maxHeight})`;
  }, [value, maxHeight, disableAutosize]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSubmit?.();
    }
    onKeyDown?.(e);
  };

  return (
    <Textarea
      ref={textareaRef}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onKeyDown={handleKeyDown}
      className={cn(
        "border-none shadow-none focus-visible:ring-0 bg-transparent text-sm resize-none min-h-[40px]",
        className,
      )}
      disabled={disabled}
      placeholder={placeholder}
      rows={1}
      {...props}
    />
  );
};

// ─── PromptInputActions ───────────────────────────────────────────────────────

const PromptInputActions: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({
  children,
  className,
  ...props
}) => (
  <div className={cn("flex items-center gap-1.5", className)} {...props}>
    {children}
  </div>
);

// ─── PromptInputAction ────────────────────────────────────────────────────────

interface PromptInputActionProps {
  tooltip: React.ReactNode;
  children: React.ReactNode;
  side?: "top" | "bottom" | "left" | "right";
  tooltipClassName?: string;
}

const PromptInputAction: React.FC<PromptInputActionProps> = ({
  tooltip,
  children,
  side = "top",
  tooltipClassName,
}) => {
  const { disabled } = usePromptInput();
  return (
    <Tooltip>
      <TooltipTrigger asChild disabled={disabled}>
        {children}
      </TooltipTrigger>
      <TooltipContent side={side} className={tooltipClassName}>
        {tooltip}
      </TooltipContent>
    </Tooltip>
  );
};

// ─── Divider ─────────────────────────────────────────────────────────────────

const CustomDivider: React.FC = () => (
  <div className="h-4 w-px bg-border mx-0.5" />
);

// ─── ToggleButton ─────────────────────────────────────────────────────────────

interface ToggleButtonProps {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
  activeClass?: string;
}

const ToggleButton: React.FC<ToggleButtonProps> = ({
  active,
  onClick,
  icon,
  label,
  activeClass = "bg-primary/10 border-primary/40 text-primary",
}) => (
  <button
    type="button"
    onClick={onClick}
    className={cn(
      "rounded-full transition-all flex items-center gap-1 px-2 py-1 border h-7 text-xs font-medium",
      active
        ? activeClass
        : "bg-transparent border-transparent text-muted-foreground hover:text-foreground hover:bg-accent/50",
    )}
  >
    <motion.div
      animate={{ rotate: active ? 360 : 0, scale: active ? 1.1 : 1 }}
      whileHover={{ scale: 1.1 }}
      transition={{ type: "spring", stiffness: 260, damping: 25 }}
    >
      {icon}
    </motion.div>
    <AnimatePresence>
      {active && (
        <motion.span
          initial={{ width: 0, opacity: 0 }}
          animate={{ width: "auto", opacity: 1 }}
          exit={{ width: 0, opacity: 0 }}
          transition={{ duration: 0.15 }}
          className="overflow-hidden whitespace-nowrap flex-shrink-0"
        >
          {label}
        </motion.span>
      )}
    </AnimatePresence>
  </button>
);

// ─── PromptInputBox ───────────────────────────────────────────────────────────

export interface PromptInputBoxProps {
  onSend?: (message: string, files?: File[]) => void;
  isLoading?: boolean;
  placeholder?: string;
  className?: string;
  onCanvasToggle?: () => void;
  isProjectLoaded?: boolean;
}

export const PromptInputBox = React.forwardRef(
  (props: PromptInputBoxProps, ref: React.Ref<HTMLDivElement>) => {
    const {
      onSend = () => {},
      isLoading = false,
      placeholder = "Describe a task or ask a question",
      className,
      onCanvasToggle,
      isProjectLoaded = false,
    } = props;

    const [input, setInput] = React.useState("");
    const [files, setFiles] = React.useState<File[]>([]);
    const [filePreviews, setFilePreviews] = React.useState<
      Record<string, string>
    >({});
    const [selectedImage, setSelectedImage] = React.useState<string | null>(
      null,
    );
    const [isRecording, setIsRecording] = React.useState(false);
    const [showSearch, setShowSearch] = React.useState(false);
    const uploadInputRef = React.useRef<HTMLInputElement>(null);
    const promptBoxRef = React.useRef<HTMLDivElement>(null);

    const isImageFile = (file: File) => file.type.startsWith("image/");

    const processFile = (file: File) => {
      if (!isImageFile(file) || file.size > 10 * 1024 * 1024) return;
      setFiles([file]);
      const reader = new FileReader();
      reader.onload = (e) =>
        setFilePreviews({ [file.name]: e.target?.result as string });
      reader.readAsDataURL(file);
    };

    const handleDragOver = React.useCallback((e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    }, []);
    const handleDragLeave = React.useCallback((e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    }, []);
    const handleDrop = React.useCallback((e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const dropped = Array.from(e.dataTransfer.files).filter(isImageFile);
      if (dropped.length > 0) processFile(dropped[0]);
    }, []);

    const handleRemoveFile = (index: number) => {
      const f = files[index];
      if (f && filePreviews[f.name]) setFilePreviews({});
      setFiles([]);
    };

    const handlePaste = React.useCallback((e: ClipboardEvent) => {
      const items = e.clipboardData?.items;
      if (!items) return;
      for (let i = 0; i < items.length; i++) {
        if (items[i].type.indexOf("image") !== -1) {
          const file = items[i].getAsFile();
          if (file) {
            e.preventDefault();
            processFile(file);
            break;
          }
        }
      }
    }, []);

    React.useEffect(() => {
      document.addEventListener("paste", handlePaste);
      return () => document.removeEventListener("paste", handlePaste);
    }, [handlePaste]);

    const handleSubmit = () => {
      if (!input.trim() && files.length === 0) return;
      const modePrefix = showSearch ? "[search]" : "";
      const formatted = [modePrefix, input].filter(Boolean).join(" ");
      onSend(formatted, files);
      setInput("");
      setFiles([]);
      setFilePreviews({});
    };

    const hasContent = input.trim() !== "" || files.length > 0;

    return (
      <>
        <PromptInput
          value={input}
          onValueChange={setInput}
          isLoading={isLoading}
          onSubmit={handleSubmit}
          className={cn(
            "w-full transition-all duration-300 ease-in-out",
            isRecording && "border-destructive/50",
            className,
          )}
          disabled={isLoading || isRecording}
          ref={ref || promptBoxRef}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          {/* File previews */}
          {files.length > 0 && !isRecording && (
            <div className="flex flex-wrap gap-2 pb-1.5 px-1">
              {files.map((file, index) => (
                <div key={index} className="relative group">
                  {file.type.startsWith("image/") &&
                    filePreviews[file.name] && (
                      <div
                        className="w-14 h-14 rounded-lg overflow-hidden cursor-pointer ring-1 ring-border"
                        onClick={() =>
                          setSelectedImage(filePreviews[file.name])
                        }
                      >
                        <img
                          src={filePreviews[file.name]}
                          alt={file.name}
                          className="h-full w-full object-cover"
                        />
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRemoveFile(index);
                          }}
                          className="absolute top-0.5 right-0.5 rounded-full bg-background/80 p-0.5"
                        >
                          <X className="h-2.5 w-2.5" />
                        </button>
                      </div>
                    )}
                </div>
              ))}
            </div>
          )}

          {/* Textarea */}
          <div
            className={cn(
              "transition-all duration-300",
              isRecording ? "h-0 overflow-hidden opacity-0" : "opacity-100",
            )}
          >
            <PromptInputTextarea
              placeholder={
                showSearch
                  ? "Search the web..."
                  : isProjectLoaded
                    ? placeholder
                    : "Select a project folder to get started..."
              }
            />
          </div>

          {/* Voice recorder */}
          {isRecording && (
            <VoiceRecorder
              isRecording={isRecording}
              onStartRecording={() => {}}
              onStopRecording={(d) => {
                setIsRecording(false);
                onSend(`[Voice message - ${d}s]`, []);
              }}
            />
          )}

          {/* Action bar */}
          <PromptInputActions
            className={cn(
              "justify-between pt-1.5",
              isRecording
                ? "opacity-0 pointer-events-none h-0 overflow-hidden"
                : "",
            )}
          >
            {/* Left tools */}
            <div className="flex items-center gap-0.5">
              {/* Attach */}
              <PromptInputAction tooltip="Attach image">
                <button
                  onClick={() => uploadInputRef.current?.click()}
                  className="flex h-7 w-7 text-muted-foreground cursor-pointer items-center justify-center rounded-md transition-colors hover:bg-accent hover:text-foreground"
                >
                  <Paperclip className="h-4 w-4" />
                  <input
                    ref={uploadInputRef}
                    type="file"
                    className="hidden"
                    onChange={(e) => {
                      if (e.target.files?.[0]) processFile(e.target.files[0]);
                      if (e.target) e.target.value = "";
                    }}
                    accept="image/*"
                  />
                </button>
              </PromptInputAction>

              <CustomDivider />

              {/* Search */}
              <ToggleButton
                active={showSearch}
                onClick={() => setShowSearch((p) => !p)}
                icon={<Globe className="w-3.5 h-3.5" />}
                label="Search"
                activeClass="bg-blue-500/10 border-blue-500/30 text-blue-500"
              />

              <CustomDivider />

              {/* Project */}
              <ToggleButton
                active={isProjectLoaded}
                onClick={() => onCanvasToggle?.()}
                icon={<FolderOpen className="w-3.5 h-3.5" />}
                label="Project"
                activeClass="bg-emerald-500/10 border-emerald-500/30 text-emerald-500"
              />
            </div>

            {/* Send / Mic / Stop */}
            <PromptInputAction
              tooltip={
                isLoading
                  ? "Stop"
                  : isRecording
                    ? "Stop recording"
                    : hasContent
                      ? "Send"
                      : "Voice input"
              }
            >
              <Button
                variant={hasContent ? "default" : "ghost"}
                size="icon"
                className={cn(
                  "h-7 w-7 rounded-full transition-all duration-200",
                  isRecording && "text-destructive hover:text-destructive",
                )}
                onClick={() => {
                  if (isRecording) setIsRecording(false);
                  else if (hasContent) handleSubmit();
                  else setIsRecording(true);
                }}
                disabled={isLoading && !hasContent}
              >
                {isLoading ? (
                  <Square className="h-3.5 w-3.5 fill-current animate-pulse" />
                ) : isRecording ? (
                  <StopCircle className="h-4 w-4" />
                ) : hasContent ? (
                  <ArrowUp className="h-3.5 w-3.5" />
                ) : (
                  <Mic className="h-4 w-4" />
                )}
              </Button>
            </PromptInputAction>
          </PromptInputActions>
        </PromptInput>

        <ImageViewDialog
          imageUrl={selectedImage}
          onClose={() => setSelectedImage(null)}
        />
      </>
    );
  },
);
PromptInputBox.displayName = "PromptInputBox";
