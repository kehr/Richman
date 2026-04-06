import { routing } from "@/domain/i18n/config";
import createMiddleware from "next-intl/middleware";

export default createMiddleware(routing);

export const config = {
	matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
