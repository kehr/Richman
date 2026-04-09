import "@testing-library/jest-dom/vitest";
import { server } from "@/test/server";

class MemoryStorage implements Storage {
	private store = new Map<string, string>();
	get length() {
		return this.store.size;
	}
	clear() {
		this.store.clear();
	}
	getItem(key: string) {
		const value = this.store.get(key);
		return value === undefined ? null : value;
	}
	key(index: number) {
		return Array.from(this.store.keys())[index] ?? null;
	}
	removeItem(key: string) {
		this.store.delete(key);
	}
	setItem(key: string, value: string) {
		this.store.set(key, value);
	}
}

const storage = new MemoryStorage();
Object.defineProperty(window, "localStorage", { value: storage });
Object.defineProperty(window, "sessionStorage", { value: storage });

// jsdom does not implement matchMedia, but antd's responsive Grid components
// (Row, Col) subscribe to it at mount. We provide a no-op polyfill that never
// matches so every breakpoint observer resolves synchronously.
if (typeof window.matchMedia !== "function") {
	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: (query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: () => {},
			removeListener: () => {},
			addEventListener: () => {},
			removeEventListener: () => {},
			dispatchEvent: () => false,
		}),
	});
}

// framer-motion schedules animation frames that write latestValues into
// element.style during an rAF callback. When a motion element unmounts
// while a frame is still queued, the callback fires and tries to write an
// undefined value into cssstyle, which crashes jsdom with a
// "Cannot read properties of undefined (reading 'split')" uncaughtException.
// The crash is async and happens after the test assertions resolve, so it
// does not affect correctness — but vitest flags the uncaught exception
// and fails the run. Swallow that exact error shape so legitimate errors
// still surface.
process.on("uncaughtException", (err: unknown) => {
	if (
		err instanceof TypeError &&
		err.message.includes("Cannot read properties of undefined (reading 'split')") &&
		err.stack?.includes("cssstyle")
	) {
		return;
	}
	throw err;
});

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
