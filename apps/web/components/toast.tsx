"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Icon } from "./icon";

type Kind = "ok" | "bad" | "warn";

interface Toast {
  id: number;
  kind: Kind;
  message: string;
  leaving: boolean;
}

interface ToastApi {
  /** Confirm that something finished, e.g. after a save. */
  success(message: string): void;
  /** Something failed. */
  error(message: string): void;
  /** Something needs attention but did not fail. */
  warn(message: string): void;
}

const ToastContext = createContext<ToastApi | null>(null);

// A failure is worth a second look, so it stays longer than a confirmation.
const AUTO_DISMISS_MS: Record<Kind, number> = {
  ok: 4000,
  bad: 8000,
  warn: 8000,
};

const EXIT_MS = 250;

/**
 * The app's feedback channel.
 *
 * Areas that used to render a banner next to the thing they described now call
 * one of these instead, so the message appears where the eye already is and does
 * not push the page around. Toasts stack bottom-right and every one can be
 * dismissed by hand, which matters when the text is a code and a request id.
 */
export function useToast(): ToastApi {
  const api = useContext(ToastContext);
  if (!api) throw new Error("useToast must be used inside a ToastProvider");
  return api;
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);
  const nextId = useRef(1);
  const timers = useRef(new Map<number, ReturnType<typeof setTimeout>>());

  const remove = useCallback((id: number) => {
    const timer = timers.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timers.current.delete(id);
    }
    // Let the exit animation play before the row is dropped.
    setItems((current) => current.map((t) => (t.id === id ? { ...t, leaving: true } : t)));
    setTimeout(() => setItems((current) => current.filter((t) => t.id !== id)), EXIT_MS);
  }, []);

  const push = useCallback(
    (kind: Kind, message: string) => {
      const id = nextId.current++;
      setItems((current) => [...current, { id, kind, message, leaving: false }]);
      timers.current.set(
        id,
        setTimeout(() => remove(id), AUTO_DISMISS_MS[kind])
      );
    },
    [remove]
  );

  useEffect(() => {
    const pending = timers.current;
    return () => {
      for (const timer of pending.values()) clearTimeout(timer);
    };
  }, []);

  const api = useMemo<ToastApi>(
    () => ({
      success: (message) => push("ok", message),
      error: (message) => push("bad", message),
      warn: (message) => push("warn", message),
    }),
    [push]
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="toasts" role="region" aria-label="Notifications">
        {items.map((toast) => (
          <div
            key={toast.id}
            className={`toast ${toast.kind}${toast.leaving ? " out" : ""}`}
            role={toast.kind === "bad" ? "alert" : "status"}
          >
            <Icon name={toast.kind === "ok" ? "check" : "alert"} />
            <div>{toast.message}</div>
            <button
              type="button"
              className="x"
              aria-label="Dismiss"
              onClick={() => remove(toast.id)}
            >
              <Icon name="x" size="s" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
