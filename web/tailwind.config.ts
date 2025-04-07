/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "sans-serif"],
      },
      colors: {
        "hyper-blue": "#1A2A44",
        "hyper-cyan": "#17A2B8",
        "hyper-gray": "#F5F6FA",
        primary: {
          50: "#E8EBF0",
          100: "#D0D7E2",
          200: "#A0AFC4",
          300: "#7187A7",
          400: "#415F89",
          500: "#1A2A44",
          600: "#15223A",
          700: "#111A2F",
          800: "#0C1325",
          900: "#080C1A",
        },
        secondary: {
          50: "#E6F7FA",
          100: "#CCEFF5",
          200: "#99DFEB",
          300: "#66CFE1",
          400: "#33BFD7",
          500: "#17A2B8",
          600: "#148293",
          700: "#10616F",
          800: "#0B414A",
          900: "#072025",
        },
        gray: {
          50: "#FFFFFF",
          100: "#F5F6FA",
          200: "#E9EBF3",
          300: "#D8DCE8",
          400: "#C8CDDD",
          500: "#B7BED1",
          600: "#949CB7",
          700: "#717A9E",
          800: "#4F5871",
          900: "#2D3243",
        },
      },
    },
  },
  plugins: [],
};
