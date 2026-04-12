import { resolve } from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [react()],
	resolve: {
		alias: {
			"@": resolve(__dirname, "src"),
		},
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks: {
					"chart-lightweight": ["lightweight-charts"],
					"chart-echarts": ["echarts", "echarts-for-react"],
				},
			},
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
