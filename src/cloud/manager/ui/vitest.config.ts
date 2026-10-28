import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    include: ["src/**/*.spec.{ts,tsx}"],
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/testing/setup.ts"],
    env: {
      VITE_API_URL: "http://api.test",
    },
  },
});
