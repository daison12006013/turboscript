/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/frontend/**/*.{js,ts,jsx,tsx,html}",
    "./app/frontend/**/*.html",
  ],
  darkMode: 'class', // Enable dark mode with class strategy
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
