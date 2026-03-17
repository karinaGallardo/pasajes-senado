// =============================================================================
// 🎨 ENTERPRISE DESIGN TOKENS (Base)
// Cambia esto para cambiar la identidad visual completa del sistema.
// =============================================================================
const PALETTE = {
  cinnabar: { 50: "#fdf4f3", 100: "#fce7e4", 500: "#e1503d", 600: "#d04432", 700: "#af3626", 800: "#913023", 950: "#41140e" },
  jaffa: { 50: "#fef7ee", 100: "#fdedd7", 300: "#f8ba79", 600: "#e15c15", 950: "#411509" },
  plum: { 50: "#fbf7fb", 100: "#f8eff7", 500: "#b972b2", 600: "#9a5391" },
  lead: { 50: "#f7f6f6", 100: "#eeedec", 200: "#dedbd9", 500: "#5C5753", 800: "#262320" },
  success: { 50: "#ecfdf5", 500: "#10b981", 600: "#059669" },
  warning: { 50: "#fffbeb", 500: "#f59e0b", 600: "#d97706" },
  danger: { 50: "#fef2f2", 500: "#ef4444", 600: "#dc2626" },
  accents: { gold: "#F29A2E", salmon: "#F2884B", bright: "#F2490C" },
};

tailwind.config = {
  theme: {
    colors: {
      transparent: "transparent",
      current: "currentColor",
      black: "#000000",
      white: "#ffffff",

      // 1. BRAND (Identidad)
      primary: {
        ...PALETTE.cinnabar,
        DEFAULT: PALETTE.cinnabar[800],
      },
      secondary: {
        ...PALETTE.jaffa,
        DEFAULT: PALETTE.jaffa[600],
      },

      // 2. ACCENTS (Énfasis)
      accent: {
        soft: PALETTE.accents.gold,
        warm: PALETTE.accents.salmon,
        bright: PALETTE.accents.bright,
      },

      // 3. NEUTRALS & SURFACE (Jerarquía)
      neutral: {
        ...PALETTE.lead,
        surface: {
          app: PALETTE.lead[50],
          card: "#ffffff",
          sidebar: PALETTE.jaffa[950],
          muted: PALETTE.lead[100],
        },
        content: {
          main: PALETTE.lead[800],
          muted: PALETTE.lead[500],
          inverse: "#ffffff",
        },
      },

      // 4. STATUS (Feedback funcional)
      status: {
        info: { ...PALETTE.plum, DEFAULT: PALETTE.plum[600] },
        success: { ...PALETTE.success, DEFAULT: PALETTE.success[500] },
        warning: { ...PALETTE.warning, DEFAULT: PALETTE.warning[500] },
        danger: { ...PALETTE.danger, DEFAULT: PALETTE.danger[500] },
      },

      // 5. INTERACTIVE (Acciones)
      action: {
        hover: "rgba(0, 0, 0, 0.05)",
        active: "rgba(0, 0, 0, 0.1)",
        disabled: PALETTE.lead[200],
      },

      // ALIASES (Compatibilidad total)
      info: { ...PALETTE.plum, DEFAULT: PALETTE.plum[600] },
      success: { ...PALETTE.success, DEFAULT: PALETTE.success[500] },
      warning: { ...PALETTE.warning, DEFAULT: PALETTE.warning[500] },
      danger: { ...PALETTE.danger, DEFAULT: PALETTE.danger[500] },
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
      "2xl": "24px",
      full: "9999px",
    },
    borderColor: function (theme) {
      return Object.assign({}, theme("colors"), {
        DEFAULT: theme("colors.neutral.200", "#e7e7e7"),
      });
    },
    ringColor: function (theme) {
      return Object.assign({}, theme("colors"), {
        DEFAULT: theme("colors.primary.DEFAULT"),
      });
    },
    transitionTimingFunction: {
      DEFAULT: "cubic-bezier(0.4, 0, 0.2, 1)",
      premium: "cubic-bezier(0.34, 1.56, 0.64, 1)",
    },
    transitionDuration: {
      DEFAULT: "200ms",
      fast: "150ms",
      slow: "400ms",
    },
    boxShadow: {
      sm: "0 1px 2px 0 rgba(0, 0, 0, 0.05)",
      DEFAULT: "0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06)",
      md: "0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)",
      soft: "0 4px 20px -2px rgba(0, 0, 0, 0.05), 0 2px 12px -2px rgba(0, 0, 0, 0.03)",
      premium: "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)",
    },
    extend: {
      fontSize: { xxs: "0.625rem", tiny: "0.5rem" },
      animation: {
        "fade-in-down": "fadeInDown 0.3s ease-out",
        "fade-in": "fadeIn 0.2s ease-in",
      },
      keyframes: {
        fadeInDown: {
          "0%": { opacity: "0", transform: "translateY(-10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        fadeIn: { "0%": { opacity: "0" }, "100%": { opacity: "1" } },
      },
    },
  },
};
