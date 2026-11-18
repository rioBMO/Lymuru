function parseVersion(version: string): number[] {
    const core = version.trim().replace(/^v/i, "").split(/[-+]/)[0];
    return core.split(".").map((part) => Number.parseInt(part, 10) || 0);
}

/**
 * Compares dotted versions numerically ("1.10.0" > "1.9.2"). A leading "v" and
 * pre-release/build suffixes ("-beta", "+sha") are ignored; missing parts count
 * as 0. Returns a negative number, 0, or a positive number.
 */
export function compareVersions(a: string, b: string): number {
    const pa = parseVersion(a);
    const pb = parseVersion(b);
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
        const diff = (pa[i] ?? 0) - (pb[i] ?? 0);
        if (diff !== 0) {
            return diff;
        }
    }
    return 0;
}

/** True when `candidate` is a strictly newer version than `current`. */
export function isNewerVersion(candidate: string, current: string): boolean {
    return compareVersions(candidate, current) > 0;
}
