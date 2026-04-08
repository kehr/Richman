import { resolveReturnTo } from "@/domain/auth/resolve-return-to";
import { LoginForm } from "@/features/auth";
import { useSearchParams } from "react-router";
import { AuthSplitLayout } from "./components/AuthSplitLayout";

// LoginPage composes the split layout + login form and plumbs the validated
// ?returnTo= path through to useLogin via the form's redirectTo prop. The
// security-critical resolveReturnTo helper lives in domain/auth so both
// LoginPage and RegisterPage can share the same validation.
export { resolveReturnTo } from "@/domain/auth/resolve-return-to";

export default function LoginPage() {
	const [searchParams] = useSearchParams();
	const redirectTo = resolveReturnTo(searchParams.get("returnTo"));

	return <AuthSplitLayout form={<LoginForm redirectTo={redirectTo} />} />;
}
