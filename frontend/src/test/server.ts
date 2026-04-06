import { setupServer } from "msw/node";

// Global MSW server used by the test setup. Individual tests register their
// own handlers via server.use(...) so this base instance stays empty.
export const handlers = [];

export const server = setupServer(...handlers);
