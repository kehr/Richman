import { resolveReturnTo } from "@/domain/auth/resolve-return-to";
import { RegisterForm } from "@/features/auth";
import { useSearchParams } from "react-router";
import { AuthSplitLayout } from "./components/AuthSplitLayout";

// RegisterPage mirrors LoginPage: it reads and validates ?returnTo= via
// the shared domain helper so a user who pivoted from the login deep
// link still ends up at the original target after registering.
// It also reads ?ref= and passes the invite code to RegisterForm so that
// shared links (e.g. /register?ref=RM3K9X7HAB) auto-fill the invite code.
export default function RegisterPage() {
	const [searchParams] = useSearchParams();
	const redirectTo = resolveReturnTo(searchParams.get("returnTo"));
	const refCode = searchParams.get("ref");

	return <AuthSplitLayout form={<RegisterForm redirectTo={redirectTo} refCode={refCode} />} />;
}
