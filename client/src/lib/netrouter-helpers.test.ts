/**
 * Design reminder — validation makes native-style error states deterministic
 * and prevents ambiguous connection feedback in the compact UI.
 */
import { describe, expect, it } from "vitest";
import { connectionError, isValidConnectionTarget } from "./netrouter-helpers";

describe("connection target validation", () => {
  it("accepts valid IPv4 router targets", () => {
    expect(isValidConnectionTarget("192.168.88.1")).toBe(true);
    expect(isValidConnectionTarget("10.0.0.254")).toBe(true);
  });

  it("accepts valid L2 MAC router targets", () => {
    expect(isValidConnectionTarget("AA:BB:CC:DD:EE:01")).toBe(true);
    expect(isValidConnectionTarget("aa:bb:cc:dd:ee:01")).toBe(true);
  });

  it("rejects malformed and out-of-range targets", () => {
    expect(isValidConnectionTarget("192.168.88.999")).toBe(false);
    expect(isValidConnectionTarget("router.local")).toBe(false);
    expect(isValidConnectionTarget("AA:BB:CC:DD:EE")).toBe(false);
  });
});

describe("connection error state", () => {
  it("returns a precise address error before checking credentials", () => {
    expect(connectionError("invalid", "")).toBe("Invalid IP address or MAC address.");
  });

  it("returns an explicit authentication error when the target is valid", () => {
    expect(connectionError("192.168.88.1", "")).toBe("Authentication failed: enter a username.");
  });

  it("allows a valid target with a username", () => {
    expect(connectionError("AA:BB:CC:DD:EE:01", "admin")).toBeNull();
  });
});
