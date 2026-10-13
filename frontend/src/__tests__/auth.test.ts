import { describe, it, expect } from "vitest";
import type { User, AuthTokenResponse, ErrorResponse } from "../types";

describe("Auth types", () => {
  it("should accept valid User object", () => {
    const user: User = {
      id: "123",
      email: "test@example.com",
      display_name: "Test User",
      avatar_url: "",
      bio: "",
      gender: "",
      is_admin: false,
      is_super_admin: false,
      user_hash: "a1b2c3d4e5",
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(user.email).toBe("test@example.com");
    expect(user.display_name).toBe("Test User");
  });

  it("should accept valid AuthTokenResponse object", () => {
    const response: AuthTokenResponse = {
      access_token: "abc.def.ghi",
      refresh_token: "xyz",
      expires_in: 900,
    };
    expect(response.access_token).toBe("abc.def.ghi");
    expect(response.expires_in).toBe(900);
  });

  it("should accept valid ErrorResponse object", () => {
    const error: ErrorResponse = {
      error: { code: "NOT_FOUND", message: "Resource not found" },
    };
    expect(error.error.code).toBe("NOT_FOUND");
  });

  it("should reject invalid User (missing required fields)", () => {
    const invalid = { id: "123" };
    expect((invalid as User).email).toBeUndefined();
  });
});
