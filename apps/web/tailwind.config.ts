import type { Config } from "tailwindcss";

const config = {
  darkMode: ["class"],
  content: [
    "./src/app/**/*.{ts,tsx}",
    "./src/components/**/*.{ts,tsx}",
    "./src/features/**/*.{ts,tsx}",
    "./src/lib/**/*.{ts,tsx}"
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-geist-sans)", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "SFMono-Regular", "monospace"]
      },
      fontSize: {
        "2xs": ["10px", { lineHeight: "1.4", letterSpacing: "0.04em" }]
      },
      colors: {
        // shadcn aliases (HSL-driven), repointed to v2 tokens:
        //   background = bg-page (#0b0e0f)
        //   foreground = text-primary (#ffffff)
        //   card       = bg-panel (#191b1f)
        //   primary    = accent green (#00e701) — button surface
        //   muted-foreground = text-secondary (#9ca3af)
        //   border     = border-subtle (#24272c)
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))"
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))"
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))"
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))"
        },
        accent: {
          DEFAULT: "var(--color-accent)",
          hover: "var(--color-accent-hover)",
          muted: "var(--color-accent-muted)",
          foreground: "var(--color-text-on-accent)"
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))"
        },

        // v2-only utilities that don't map cleanly to shadcn semantics.
        page: "var(--color-bg-page)",
        panel: "var(--color-bg-panel)",
        elevated: "var(--color-bg-elevated)",
        "border-strong": "var(--color-border-strong)",
        faint: "var(--color-text-muted)", // #474f54 low-emphasis text
        danger: "var(--color-danger)",
        warning: "var(--color-warning)"
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)"
      }
    }
  },
  plugins: [require("tailwindcss-animate")]
} satisfies Config;

export default config;
