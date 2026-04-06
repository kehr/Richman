import react from "@vitejs/plugin-react";
import { resolve } from "node:path";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [react()],
	resolve: {
		alias: {
			"@": resolve(__dirname, "src"),
		},
	},
	server: {
		port: 3000,
		proxy: {
			"/api": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
		},
	},
});
