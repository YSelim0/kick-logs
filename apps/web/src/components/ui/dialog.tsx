"use client";

import { X } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";

export function Dialog({
  open,
  onOpenChange,
  children
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onOpenChange(false);
    };
    if (open) document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onOpenChange]);

  if (!open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onOpenChange(false);
      }}
      style={{ backgroundColor: "rgba(0,0,0,0.7)" }}
    >
      {children}
    </div>,
    document.body
  );
}

export function DialogContent({
  children,
  className = ""
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`relative w-full rounded-lg border bg-black p-6 shadow-lg ${className}`}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>
  );
}

export function DialogHeader({ children }: { children: ReactNode }) {
  return <div className="mb-4 space-y-1">{children}</div>;
}

export function DialogTitle({
  children,
  className = ""
}: {
  children: ReactNode;
  className?: string;
}) {
  return <h2 className={`text-base font-semibold ${className}`}>{children}</h2>;
}

export function DialogDescription({
  children,
  className = ""
}: {
  children: ReactNode;
  className?: string;
}) {
  return <p className={`text-sm ${className}`}>{children}</p>;
}

export function DialogClose({ onClose }: { onClose: () => void }) {
  return (
    <button
      className="absolute right-4 top-4 text-muted-foreground hover:text-foreground"
      onClick={onClose}
      type="button"
    >
      <X className="h-4 w-4" />
    </button>
  );
}
