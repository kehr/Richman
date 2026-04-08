import { Typography } from "@/ui-kit/eat";
import type { ReactNode } from "react";
import { SampleDecisionCard } from "./SampleDecisionCard";

const { Title, Paragraph, Text } = Typography;

interface AuthSplitLayoutProps {
	form: ReactNode;
}

// AuthSplitLayout is the responsive 2-column shell shared by LoginPage and
// RegisterPage. It implements the breakpoints from PRD §2.1:
//   - >= 1200px: left 60% / right 40%
//   - 1024-1199px: left 50% / right 50%
//   - <  1024px: single column, left content stacked above the form
//
// The layout uses CSS Grid via inline style + a small <style> block so we
// don't need to add a new global stylesheet or pull in antd's Grid (which
// doesn't expose true CSS-driven percentage breakpoints without JS).
export function AuthSplitLayout({ form }: AuthSplitLayoutProps) {
	return (
		<div className="auth-split-layout" data-testid="auth-split-layout">
			<style>{AUTH_SPLIT_LAYOUT_CSS}</style>
			<section className="auth-split-layout__left" data-testid="auth-split-layout-left">
				<div className="auth-split-layout__brand">
					<Title level={2} style={{ marginBottom: 8 }}>
						Richman
					</Title>
					<Text type="secondary" style={{ fontSize: 16 }}>
						把基金经理的思维方式装进你的口袋
					</Text>
				</div>
				<div className="auth-split-layout__intro">
					<Paragraph style={{ marginBottom: 8 }}>
						基于你的真实持仓，每天给出可执行的建议，而不是另一份新闻摘要。
					</Paragraph>
					<Paragraph style={{ marginBottom: 8 }}>
						每张决策卡同时回答三个问题：趋势怎么样？我的位置如何？有没有催化剂？
					</Paragraph>
					<Paragraph style={{ marginBottom: 0 }}>
						建议本身明确到动作和仓位，并标注信心度，方便你判断该不该照做。
					</Paragraph>
				</div>
				<div className="auth-split-layout__sample">
					<SampleDecisionCard />
				</div>
			</section>
			<section className="auth-split-layout__right" data-testid="auth-split-layout-right">
				<div className="auth-split-layout__form-wrapper">{form}</div>
			</section>
		</div>
	);
}

// AUTH_SPLIT_LAYOUT_CSS keeps the responsive behavior fully in CSS so we can
// guarantee the desktop split is server-side-stable and not subject to JS
// resize listeners. The class names are scoped via the `auth-split-layout`
// prefix to avoid leaking into the rest of the app.
const AUTH_SPLIT_LAYOUT_CSS = `
.auth-split-layout {
	display: grid;
	grid-template-columns: 1fr;
	min-height: 100vh;
	width: 100%;
}
.auth-split-layout__left {
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: 24px;
	padding: 48px 32px;
	background: #fafafa;
}
.auth-split-layout__brand {
	max-width: 480px;
}
.auth-split-layout__intro {
	max-width: 480px;
}
.auth-split-layout__sample {
	max-width: 480px;
}
.auth-split-layout__right {
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 48px 32px;
}
.auth-split-layout__form-wrapper {
	width: 100%;
	max-width: 360px;
}
@media (min-width: 1024px) and (max-width: 1199px) {
	.auth-split-layout {
		grid-template-columns: 1fr 1fr;
	}
}
@media (min-width: 1200px) {
	.auth-split-layout {
		grid-template-columns: 6fr 4fr;
	}
}
`;
