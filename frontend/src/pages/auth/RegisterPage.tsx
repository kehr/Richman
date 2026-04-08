import { RegisterForm } from "@/features/auth";
import { AuthSplitLayout } from "./components/AuthSplitLayout";

export default function RegisterPage() {
	return <AuthSplitLayout form={<RegisterForm />} />;
}
