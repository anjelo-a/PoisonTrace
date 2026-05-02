import { describe, expect, it, vi } from "vitest";
import { formatDateTime, shortAddress, timeAgo } from "../app/lib/format";

describe("format helpers", () => {
  it("shortAddress keeps head/tail", () => {
    expect(shortAddress("ABCDEFGH12345678", 4, 4)).toBe("ABCD...5678");
  });

  it("formatDateTime returns UTC formatted string", () => {
    expect(formatDateTime("2026-05-02T10:00:00Z")).toContain("2026-05-02 10:00:00");
  });

  it("timeAgo shows minutes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-02T10:10:00Z"));
    expect(timeAgo("2026-05-02T10:05:00Z")).toBe("5m ago");
    vi.useRealTimers();
  });
});
