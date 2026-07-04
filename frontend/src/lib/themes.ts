export interface ThemeTokens {
  /* shadcn-style tokens */
  background: string;
  foreground: string;
  card: string;
  "card-foreground": string;
  popover: string;
  "popover-foreground": string;
  primary: string;
  "primary-foreground": string;
  secondary: string;
  "secondary-foreground": string;
  muted: string;
  "muted-foreground": string;
  accent: string;
  "accent-foreground": string;
  destructive: string;
  "destructive-foreground": string;
  border: string;
  input: string;
  ring: string;
  radius: string;

  /* legacy aliases (kept for backward compatibility, remove after migration) */
  "primary-dark": string;
  "primary-glow": string;
  "primary-gradient": string;
  bg: string;
  "bg-soft": string;
  "bg-card": string;
  text: string;
  "text-secondary": string;
  "text-muted": string;
  "border-light": string;
  "destructive-light": string;
  success: string;
  "success-light": string;
  "radius-sm": string;
  "radius-md": string;
  "radius-lg": string;
  "radius-xl": string;
  "shadow-sm": string;
  "shadow-md": string;
  "shadow-lg": string;
  "shadow-glow": string;
}

const lightTokens = {
  /* shadcn */
  background: "#FFFFFF",
  foreground: "#0C1A2E",
  card: "#FFFFFF",
  "card-foreground": "#0C1A2E",
  popover: "#FFFFFF",
  "popover-foreground": "#0C1A2E",
  primary: "#4FC3F7",
  "primary-foreground": "#FFFFFF",
  secondary: "#F0F9FF",
  "secondary-foreground": "#0C1A2E",
  muted: "#F0F4F8",
  "muted-foreground": "#6B7B8C",
  accent: "#E0F5FE",
  "accent-foreground": "#0C1A2E",
  destructive: "#E53935",
  "destructive-foreground": "#FFFFFF",
  border: "#E0E8F0",
  input: "#E0E8F0",
  ring: "#4FC3F7",
  radius: "0.625rem",

  /* legacy aliases */
  "primary-dark": "#0288D1",
  "primary-glow": "rgba(79, 195, 247, 0.25)",
  "primary-gradient": "linear-gradient(135deg, #4FC3F7 0%, #0288D1 100%)",
  bg: "#FFFFFF",
  "bg-soft": "#F0F4F8",
  "bg-card": "#FFFFFF",
  text: "#0C1A2E",
  "text-secondary": "#6B7B8C",
  "text-muted": "#8899AA",
  "border-light": "#F0F4F8",
  "destructive-light": "#FFF0F0",
  success: "#43A047",
  "success-light": "#F0FFF0",
  "radius-sm": "calc(0.625rem - 4px)",
  "radius-md": "calc(0.625rem - 2px)",
  "radius-lg": "0.625rem",
  "radius-xl": "calc(0.625rem + 4px)",
  "shadow-sm": "0 1px 3px rgba(12,26,46,0.06)",
  "shadow-md": "0 4px 12px rgba(12,26,46,0.08)",
  "shadow-lg": "0 8px 24px rgba(12,26,46,0.10)",
  "shadow-glow": "0 0 20px rgba(79,195,247,0.15)",
} satisfies ThemeTokens;

const darkTokens = {
  /* shadcn */
  background: "#0D1117",
  foreground: "#E6EDF3",
  card: "#161B22",
  "card-foreground": "#E6EDF3",
  popover: "#161B22",
  "popover-foreground": "#E6EDF3",
  primary: "#4FC3F7",
  "primary-foreground": "#0C1A2E",
  secondary: "#1C2530",
  "secondary-foreground": "#E6EDF3",
  muted: "#21262D",
  "muted-foreground": "#8B949E",
  accent: "#1C2530",
  "accent-foreground": "#E6EDF3",
  destructive: "#F85149",
  "destructive-foreground": "#FFFFFF",
  border: "#30363D",
  input: "#30363D",
  ring: "#4FC3F7",
  radius: "0.625rem",

  /* legacy aliases */
  "primary-dark": "#81D4FA",
  "primary-glow": "rgba(79, 195, 247, 0.20)",
  "primary-gradient": "linear-gradient(135deg, #4FC3F7 0%, #0288D1 100%)",
  bg: "#0D1117",
  "bg-soft": "#161B22",
  "bg-card": "#161B22",
  text: "#E6EDF3",
  "text-secondary": "#8B949E",
  "text-muted": "#6E7681",
  "border-light": "#21262D",
  "destructive-light": "#FFEBE9",
  success: "#3FB950",
  "success-light": "#EFFFEA",
  "radius-sm": "calc(0.625rem - 4px)",
  "radius-md": "calc(0.625rem - 2px)",
  "radius-lg": "0.625rem",
  "radius-xl": "calc(0.625rem + 4px)",
  "shadow-sm": "0 1px 3px rgba(0,0,0,0.30)",
  "shadow-md": "0 4px 12px rgba(0,0,0,0.40)",
  "shadow-lg": "0 8px 24px rgba(0,0,0,0.50)",
  "shadow-glow": "0 0 20px rgba(79,195,247,0.12)",
} satisfies ThemeTokens;

export const light: ThemeTokens = lightTokens;
export const dark: ThemeTokens = darkTokens;

let _current: ThemeTokens = light;

export function getTheme(): ThemeTokens {
  return _current;
}

export function applyTheme(mode: "light" | "dark"): void {
  const tokens = mode === "dark" ? dark : light;
  _current = tokens;
  const root = document.documentElement;
  for (const [key, value] of Object.entries(tokens)) {
    root.style.setProperty(`--${key}`, value);
  }
  if (mode === "dark") {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
  try {
    localStorage.setItem("lymuru_theme", mode);
  } catch {
    /* noop */
  }
}

export function initTheme(): void {
  try {
    const saved = localStorage.getItem("lymuru_theme");
    applyTheme(saved === "dark" ? "dark" : "light");
  } catch {
    applyTheme("light");
  }
}
