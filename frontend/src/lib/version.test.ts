import { describe, expect, test } from "bun:test";
import { compareVersions, isNewerVersion } from "./version";

describe("compareVersions", () => {
    test("orders parts numerically, not as strings", () => {
        expect(compareVersions("1.10.0", "1.9.0")).toBeGreaterThan(0);
        expect(compareVersions("1.9.0", "1.10.0")).toBeLessThan(0);
        expect(compareVersions("2.0.0", "1.99.99")).toBeGreaterThan(0);
    });

    test("treats equal versions as equal", () => {
        expect(compareVersions("1.1.0", "1.1.0")).toBe(0);
    });

    test("ignores a leading v", () => {
        expect(compareVersions("v1.1.0", "1.1.0")).toBe(0);
        expect(compareVersions("V1.2.0", "v1.1.0")).toBeGreaterThan(0);
    });

    test("treats missing parts as zero", () => {
        expect(compareVersions("1.1", "1.1.0")).toBe(0);
        expect(compareVersions("1.1.1", "1.1")).toBeGreaterThan(0);
    });

    test("ignores pre-release and build suffixes", () => {
        expect(compareVersions("1.2.0-beta", "1.2.0")).toBe(0);
        expect(compareVersions("1.2.0+abc123", "1.1.9")).toBeGreaterThan(0);
    });
});

describe("isNewerVersion", () => {
    test("is true only for strictly newer versions", () => {
        expect(isNewerVersion("1.1.0", "1.0.0")).toBe(true);
        expect(isNewerVersion("1.1.0", "1.1.0")).toBe(false);
        expect(isNewerVersion("1.0.3", "1.1.0")).toBe(false);
    });
});
