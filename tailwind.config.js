/** @type {import('tailwindcss').Config} */

module.exports = {
  content: ["./web/templates/**/*.html", "./web/static/js/**/*.js"],
  theme: {
    extend: {
      colors: {
        // 1. PRIMARY (Rojo Cinnabar - HSL 7 61% 35%)
        primary: {
          50: "#fdf2f1",
          100: "#fce8e6",
          200: "#fad1cc",
          300: "#f6aba3",
          400: "#f07b70",
          500: "#e65042",
          600: "#d33a2d",
          DEFAULT: "#893026", // Tu color base exacto
          700: "#893026",
          800: "#722a21",
          900: "#5f261e",
          950: "#34110d",
        },
        // 2. SECONDARY (Naranja Jaffa - HSL 21 85% 48%)
        secondary: {
          50: "#fff7ed",
          100: "#ffeed4",
          200: "#ffdaa9",
          300: "#ffbd71",
          400: "#ff9231",
          500: "#f97316",
          DEFAULT: "#e8620b", // Tu color secundario exacto
          600: "#e8620b",
          700: "#c0490b",
          800: "#993a10",
          900: "#7b3111",
          950: "#431606",
        },
        // 3. ACCENT (Azul Pizarra - HSL 222 47% 31%)
        accent: {
          50: "#f1f5f9",
          100: "#e2e8f0",
          200: "#cbd5e1",
          300: "#94a3b8",
          400: "#64748b",
          500: "#475569",
          600: "#334155",
          DEFAULT: "#2a3b56", // Tu color accent exacto
          700: "#2a3b56",
          800: "#1e293b",
          900: "#0f172a",
          950: "#020617",
        },
        // 4. NEUTRAL (Plomo Cálido - HSL 30 5% 35%)
        neutral: {
          50: "#fafafa",
          100: "#f5f5f5",
          200: "#e5e5e5",
          300: "#d4d4d4",
          400: "#a3a3a3",
          500: "#737373",
          600: "#525252",
          DEFAULT: "#525252",
          700: "#404040",
          800: "#262626",
          900: "#171717",
          950: "#0a0a0a",
        },
        // 5. STATUS
        success: {
          50: "#ecfdf5",
          100: "#d1fae5",
          500: "#10b981",
          DEFAULT: "#0f7654", // Tu success base
          600: "#059669",
        },
        warning: {
          50: "#fffbeb",
          100: "#fef3c7",
          500: "#f59e0b",
          DEFAULT: "#d97706", // Tu warning base
          600: "#d97706",
        },
        danger: {
          50: "#fef2f2",
          100: "#fee2e2",
          500: "#ef4444",
          DEFAULT: "#dc2626", // Tu danger base
          600: "#dc2626",
        },
        info: {
          50: "#faf5ff",
          100: "#f3e8ff",
          500: "#a855f7",
          DEFAULT: "#8c2e7c", // Tu plum base
          600: "#9333ea",
        },
      },
      // Interface Tokens (Cero CSS inheritance)
      backgroundColor: {
        app: "#f9fafb",
        card: "#ffffff",
        popover: "#ffffff",
        sidebar: "#f9fafb", // primary-50 aproximado para sidebar
      },
      textColor: {
        main: "#292524", // neutral-800
        muted: "#78716c", // neutral-500
        inverse: "#ffffff",
      },
      borderColor: {
        base: "#e7e5e4", // neutral-200
        input: "#d6d3d1", // neutral-300
      },
      ringColor: {
        focus: "#fce8e6", // primary-100 para rings
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui"],
        brand: ["Montserrat", "sans-serif"],
      },
      borderRadius: {
        none: "0",
        sm: "4px",
        md: "8px",
        lg: "12px",
        xl: "16px",
        full: "9999px",
      },
      fontSize: {
        xxs: "0.625rem",
        tiny: "0.5rem",
      },
      animation: {
        "fade-in-down": "fadeInDown 0.3s ease-out",
        "fade-in": "fadeIn 0.2s ease-in",
      },
      keyframes: {
        fadeInDown: {
          "0%": { opacity: "0", transform: "translateY(-10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        fadeIn: {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
      },
      boxShadow: {
        soft: "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)",
        premium: "0 20px 25px -5px rgb(0 0 0 / 0.1), 0 10px 10px -5px rgb(0 0 0 / 0.04)",
      },
    },
  },
  plugins: [],
};
