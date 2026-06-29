import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Terminal, Lock, ArrowRight } from "lucide-react";

export type PermissionResponse = "once" | "always" | "deny";

interface PermissionDialogProps {
  open: boolean;
  action: string;
  question?: string;
  onRespond: (response: PermissionResponse) => void;
}

export function PermissionDialog({
  open,
  action,
  question,
  onRespond,
}: PermissionDialogProps) {
  return (
    <AlertDialog open={open}>
      <AlertDialogContent
        size="default"
        onEscapeKeyDown={(e) => e.preventDefault()}
        className="max-w-[400px] gap-0 p-4 overflow-hidden "
      >
        {/* Header */}
        <div className="flex items-center gap-2 px-4 py-3.5 border-b border-border">
          <div className="flex items-center justify-center w-7 h-7 rounded-md bg-warning/10 border border-warning/20 shrink-0">
            <Terminal className="size-3.5 text-warning" />
          </div>
          <AlertDialogTitle className="text-[13px] font-medium">
            Run command?
          </AlertDialogTitle>
        </div>

        <AlertDialogDescription className="sr-only">
          An agent is requesting permission to run a command in your project.
        </AlertDialogDescription>

        {/* Body */}
        <div className="flex flex-col gap-3 px-4 py-3.5">
          {question && (
            <p className="text-[13px] text-muted-foreground leading-relaxed">
              {question}
            </p>
          )}

          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2.5">
            <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wide mb-1.5">
              Command
            </p>
            <div className="flex gap-1.5 font-mono text-xs text-foreground break-all leading-relaxed">
              <span className="select-none text-muted-foreground shrink-0">
                $
              </span>
              <span>{action}</span>
            </div>
          </div>
        </div>

        {/* Footer */}
        <AlertDialogFooter className="flex items-center gap-1.5 px-4  pb-3.5 pt-4">
          <Button
            variant="ghost"
            size="sm"
            className="h-[30px] px-3 text-xs text-muted-foreground"
            onClick={() => onRespond("deny")}
          >
            Deny
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="h-[30px] px-3 text-xs gap-1.5"
            onClick={() => onRespond("always")}
          >
            <Lock className="size-3" />
            Allow always
          </Button>
          <Button
            size="sm"
            className="h-[30px] px-3 text-xs gap-1.5"
            onClick={() => onRespond("once")}
          >
            Allow once
            <ArrowRight className="size-3" />
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
