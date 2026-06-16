import { describe, it, expect, beforeEach } from "vitest";
import {
  setAuthTokens,
  clearAuthTokens,
  getAccessToken,
} from "../services/api";

describe("API token management", () => {
  beforeEach(() => {
    clearAuthTokens();
  });

  it("should start with no access token", () => {
    expect(getAccessToken()).toBeNull();
  });

  it("should store and retrieve access token", () => {
    setAuthTokens("access-123", "refresh-456");
    expect(getAccessToken()).toBe("access-123");
  });

  it("should clear tokens on clearAuthTokens", () => {
    setAuthTokens("access-123", "refresh-456");
    clearAuthTokens();
    expect(getAccessToken()).toBeNull();
  });

  it("should update access token on multiple calls", () => {
    setAuthTokens("token-v1", "refresh-v1");
    setAuthTokens("token-v2", "refresh-v2");
    expect(getAccessToken()).toBe("token-v2");
  });
});
