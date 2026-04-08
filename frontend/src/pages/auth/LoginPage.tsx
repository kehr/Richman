import { LoginForm } from "@/features/auth";
import { AuthSplitLayout } from "./components/AuthSplitLayout";

export default function LoginPage() {
	return <AuthSplitLayout form={<LoginForm />} />;
}
