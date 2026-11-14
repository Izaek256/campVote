import { ApiError } from "./api";

/**
 * One attempt key per election, kept in sessionStorage so a reload retries the
 * same submission instead of creating a second ballot (issue #37).
 */
const PREFIX = "campvote:vote-attempt:";

function session(): Storage | null {
  // sessionStorage throws in a blocked-storage browser; a missing key is
  // recoverable, a crash on the ballot page is not.
  try {
    return typeof window === "undefined" ? null : window.sessionStorage;
  } catch {
    return null;
  }
}

export function voteAttemptKey(electionId: string): string {
  const storage = session();
  const name = PREFIX + electionId;
  const stored = storage?.getItem(name);
  if (stored) return stored;
  const key =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : `attempt-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  storage?.setItem(name, key);
  return key;
}

/** Only a definite final state may clear the key. */
export function clearVoteAttemptKey(electionId: string): void {
  session()?.removeItem(PREFIX + electionId);
}

/**
 * Retries only what is safe to retry with the same key: a lost connection, a
 * timeout or a server failure. A 4xx is an answer, not a transient fault.
 */
export async function submitWithRetry<T>(send: () => Promise<T>, attempts = 3): Promise<T> {
  let delay = 400;
  for (let attempt = 1; ; attempt++) {
    try {
      return await send();
    } catch (error) {
      const retryable = !(error instanceof ApiError) || error.status >= 500 || error.status === 429;
      if (!retryable || attempt >= attempts) throw error;
      await new Promise((resolve) => setTimeout(resolve, delay + Math.random() * delay));
      delay *= 2;
    }
  }
}
