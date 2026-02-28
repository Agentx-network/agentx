/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: "#121218",
          card: "rgba(255,255,255,0.04)",
        },
        neon: {
          pink: "#FF0092",
          purple: "#AE00FF",
          cyan: "#00FFFF",
        },
      },
      boxShadow: {
        neon: "0 0 15px rgba(255,0,146,0.3)",
        "neon-cyan": "0 0 15px rgba(0,255,255,0.3)",
      },
    },
  },
  plugins: [],
};
