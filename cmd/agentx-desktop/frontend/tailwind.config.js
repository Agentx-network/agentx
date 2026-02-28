/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: "#0a0a12",
          card: "rgba(15,10,25,0.85)",
          sidebar: "#0d0b18",
        },
        neon: {
          pink: "#FF0092",
          green: "#00FF41",
          purple: "#AE00FF",
          cyan: "#00FFFF",
        },
      },
      boxShadow: {
        "neon-pink": "0 0 15px rgba(255,0,146,0.4), 0 0 40px rgba(255,0,146,0.15), inset 0 0 15px rgba(255,0,146,0.05)",
        "neon-green": "0 0 15px rgba(0,255,65,0.4), 0 0 40px rgba(0,255,65,0.15), inset 0 0 15px rgba(0,255,65,0.05)",
        "neon-cyan": "0 0 15px rgba(0,255,255,0.4), 0 0 40px rgba(0,255,255,0.15), inset 0 0 15px rgba(0,255,255,0.05)",
        "neon-purple": "0 0 15px rgba(174,0,255,0.4), 0 0 40px rgba(174,0,255,0.15)",
        neon: "0 0 15px rgba(255,0,146,0.4), 0 0 40px rgba(255,0,146,0.15)",
        "glow-pink-sm": "0 0 8px rgba(255,0,146,0.5)",
        "glow-green-sm": "0 0 8px rgba(0,255,65,0.5)",
      },
    },
  },
  plugins: [],
};
