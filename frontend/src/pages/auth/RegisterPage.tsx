import { resolveReturnTo } from "@/domain/auth/resolve-return-to";
import { RegisterForm } from "@/features/auth";
import { useSearchParams } from "react-router";
import { AuthSplitLayout } from "./components/AuthSplitLayout";

// RegisterPage mirrors LoginPage: it reads and validates ?returnTo= via
// the shared domain helper so a user who pivoted from the login deep
// link still ends up at the original target after registering.
export default function RegisterPage() {
	const [searchParams] = useSearchParams();
	const redirectTo = resolveReturnTo(searchParams.get("returnTo"));

	return <AuthSplitLayout form={<RegisterForm redirectTo={redirectTo} />} />;
}
