import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vite.dev/config/
export default defineConfig({
    plugins: [tailwindcss(), react()],
    resolve: {
        alias: {
            "@api": path.resolve(import.meta.dirname, "./src/api"),
            "@assets": path.resolve(import.meta.dirname, "./src/assets"),
            "@components": path.resolve(import.meta.dirname, "./src/components"),
            "@data": path.resolve(import.meta.dirname, "./src/data"),
            "@hooks": path.resolve(import.meta.dirname, "./src/hooks"),
            "@lib": path.resolve(import.meta.dirname, "./src/lib"),
            "@pages": path.resolve(import.meta.dirname, "./src/pages"),
            "@routes": path.resolve(import.meta.dirname, "./src/routes"),
            "@schemas": path.resolve(import.meta.dirname, "./src/schemas"),
            "@styles": path.resolve(import.meta.dirname, "./src/styles"),
        },
    },
});
