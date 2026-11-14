// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api";
import { clearVoteAttemptKey, submitWithRetry, voteAttemptKey } from "@/lib/vote-attempt";

beforeEach(() => sessionStorage.clear());
afterEach(() => vi.useRealTimers());

describe("voteAttemptKey", () => {
  it("keeps one stable key per election", () => {
    const first = voteAttemptKey("e-1");
    expect(voteAttemptKey("e-1")).toBe(first);
    expect(voteAttemptKey("e-2")).not.toBe(first);
  });

  it("issues a new key after a definite final state", () => {
    const first = voteAttemptKey("e-1");
    clearVoteAttemptKey("e-1");
    expect(voteAttemptKey("e-1")).not.toBe(first);
    expect(sessionStorage.length).toBe(1);
  });
});

describe("submitWithRetry", () => {
  it("retries a lost connection and then succeeds", async () => {
    vi.useFakeTimers();
    let calls = 0;
    const sent = submitWithRetry(async () => {
      calls += 1;
      if (calls < 3) throw new TypeError("network");
      return "ok";
    });
    await vi.runAllTimersAsync();
    await expect(sent).resolves.toBe("ok");
    expect(calls).toBe(3);
  });

  it("does not retry a definite client answer", async () => {
    const send = vi.fn(async () => {
      throw new ApiError("you have already voted", 409);
    });
    await expect(submitWithRetry(send)).rejects.toBeInstanceOf(ApiError);
    expect(send).toHaveBeenCalledTimes(1);
  });
});
