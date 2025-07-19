/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/hybrid/**/*.{js,ts,jsx,tsx,html}",
    "./app/hybrid/**/*.html",
  ],
  darkMode: 'class', // Enable dark mode with class strategy
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
