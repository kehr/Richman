import "./styles/global.css";
import { migrateStorageKeys } from "@/domain/storage/local-storage";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";

// Migrate legacy localStorage keys to the richman_ prefixed variants.
// Must run before any feature code reads from storage so that auth tokens
// and settings written by v1 clients are preserved across the upgrade.
migrateStorageKeys();

const root = document.getElementById("root");
if (root) {
	createRoot(root).render(
		<StrictMode>
			<App />
		</StrictMode>,
	);
}
