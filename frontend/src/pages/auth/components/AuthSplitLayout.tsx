import type { ReactNode } from "react";
import { SampleDecisionCard } from "./SampleDecisionCard";

interface AuthSplitLayoutProps {
	form: ReactNode;
}

// AuthSplitLayout is the responsive 2-column shell shared by LoginPage and
// RegisterPage. It implements an editorial-style split layout:
//   - Left column: right-anchored content block near the divider, large
//     display slogan, a single supporting paragraph, SampleDecisionCard as
//     the visual anchor, and a thin footer line. Background is a very
//     subtle gradient to add depth without competing with the card.
//   - Right column: form block left-anchored near the divider so both
//     columns share the divider as a visual rail.
//
// The responsive breakpoints match PRD §2.1:
//   - >= 1200px: left 60% / right 40%, content asymmetrically anchored
//   - 1024-1199px: left 50% / right 50%, same anchoring with tighter gaps
//   - <  1024px: single column stack, content re-centered
//
// The layout uses CSS Grid via inline style + a scoped <style> block so we
// don't need to add a new global stylesheet or pull in antd's Grid (which
// doesn't expose true CSS-driven percentage breakpoints without JS).
export function AuthSplitLayout({ form }: AuthSplitLayoutProps) {
	return (
		<div className="auth-split-layout" data-testid="auth-split-layout">
			<style>{AUTH_SPLIT_LAYOUT_CSS}</style>
			<section className="auth-split-layout__left" data-testid="auth-split-layout-left">
				<div className="auth-split-layout__content">
					<header className="auth-split-layout__brand">
						<img
							src="/logo.svg"
							alt="Richman logo"
							className="auth-split-layout__brand-mark"
							width={44}
							height={44}
						/>
						<span className="auth-split-layout__brand-name">Richman</span>
					</header>

					<div className="auth-split-layout__hero">
						<h1 className="auth-split-layout__slogan">
							把基金经理的思维方式
							<br />
							装进你的口袋
						</h1>
						<p className="auth-split-layout__subtitle">
							基于你的真实持仓，每天给出可执行的建议，而不是另一份新闻摘要。
						</p>
					</div>

					<div className="auth-split-layout__sample">
						<SampleDecisionCard />
					</div>
				</div>

				<div className="auth-split-layout__footer-rail">
					<div className="auth-split-layout__footer">
						<span className="auth-split-layout__footer-mark">RICHMAN</span>
						<span className="auth-split-layout__footer-sep">·</span>
						<span className="auth-split-layout__footer-tag">DECISION, NOT NEWS</span>
					</div>
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
// resize listeners. Class names are scoped via the `auth-split-layout`
// prefix to avoid leaking into the rest of the app.
const AUTH_SPLIT_LAYOUT_CSS = `
.auth-split-layout {
	display: grid;
	grid-template-columns: 1fr;
	min-height: 100vh;
	width: 100%;
	color: #0b0b0d;
	font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI",
		"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
	font-feature-settings: "ss01", "cv11";
}
.auth-split-layout__left {
	position: relative;
	display: flex;
	flex-direction: column;
	padding: 56px 0 32px;
	background:
		radial-gradient(120% 80% at 100% 0%, rgba(11, 11, 13, 0.04) 0%, transparent 55%),
		linear-gradient(180deg, #fbfbfc 0%, #f3f3f5 100%);
	overflow: hidden;
}
.auth-split-layout__left::before {
	content: "";
	position: absolute;
	inset: 0;
	background-image:
		linear-gradient(to right, rgba(11, 11, 13, 0.035) 1px, transparent 1px),
		linear-gradient(to bottom, rgba(11, 11, 13, 0.035) 1px, transparent 1px);
	background-size: 48px 48px;
	mask-image: radial-gradient(120% 100% at 70% 50%, #000 30%, transparent 75%);
	-webkit-mask-image: radial-gradient(120% 100% at 70% 50%, #000 30%, transparent 75%);
	pointer-events: none;
}
.auth-split-layout__content {
	position: relative;
	flex: 1;
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: 44px;
	margin-left: auto;
	margin-right: clamp(40px, 6vw, 96px);
	max-width: 520px;
	width: calc(100% - 80px);
	box-sizing: border-box;
}
.auth-split-layout__brand {
	display: flex;
	align-items: center;
	gap: 14px;
}
.auth-split-layout__brand-mark {
	display: block;
	border-radius: 11px;
	box-shadow: 0 6px 20px rgba(11, 11, 13, 0.18);
}
.auth-split-layout__brand-name {
	font-size: 22px;
	font-weight: 600;
	letter-spacing: -0.01em;
	color: #0b0b0d;
}
.auth-split-layout__hero {
	display: flex;
	flex-direction: column;
	gap: 20px;
}
.auth-split-layout__slogan {
	margin: 0;
	font-size: clamp(32px, 3.6vw, 52px);
	font-weight: 500;
	line-height: 1.12;
	letter-spacing: -0.02em;
	color: #0b0b0d;
}
.auth-split-layout__subtitle {
	margin: 0;
	font-size: 16px;
	line-height: 1.7;
	color: #4a4a52;
	max-width: 440px;
}
.auth-split-layout__sample {
	max-width: 480px;
}
.auth-split-layout__footer-rail {
	position: relative;
	margin-top: 40px;
}
.auth-split-layout__footer {
	position: relative;
	display: flex;
	align-items: center;
	gap: 10px;
	margin-left: auto;
	margin-right: clamp(40px, 6vw, 96px);
	max-width: 520px;
	width: calc(100% - 80px);
	padding-top: 24px;
	border-top: 1px solid rgba(11, 11, 13, 0.08);
	font-size: 11px;
	letter-spacing: 0.14em;
	color: #8e8e93;
	text-transform: uppercase;
}
.auth-split-layout__footer-mark {
	font-weight: 600;
	color: #4a4a52;
}
.auth-split-layout__footer-sep {
	opacity: 0.5;
}
.auth-split-layout__right {
	position: relative;
	display: flex;
	flex-direction: column;
	justify-content: center;
	padding: 56px 0;
	background: #ffffff;
}
.auth-split-layout__right::before {
	content: "";
	position: absolute;
	top: 0;
	bottom: 0;
	left: 0;
	width: 1px;
	background: linear-gradient(
		to bottom,
		transparent 0%,
		rgba(11, 11, 13, 0.08) 20%,
		rgba(11, 11, 13, 0.08) 80%,
		transparent 100%
	);
}
.auth-split-layout__form-wrapper {
	margin-right: auto;
	margin-left: clamp(40px, 6vw, 96px);
	max-width: 400px;
	width: calc(100% - 80px);
}
@media (min-width: 1024px) and (max-width: 1199px) {
	.auth-split-layout {
		grid-template-columns: 1fr 1fr;
	}
	.auth-split-layout__content,
	.auth-split-layout__footer {
		max-width: 440px;
		margin-right: clamp(32px, 4vw, 56px);
	}
	.auth-split-layout__form-wrapper {
		margin-left: clamp(32px, 4vw, 56px);
	}
}
@media (min-width: 1200px) {
	.auth-split-layout {
		grid-template-columns: 6fr 4fr;
	}
}
@media (max-width: 1023px) {
	.auth-split-layout__left {
		padding: 48px 0 24px;
	}
	.auth-split-layout__content {
		margin-left: auto;
		margin-right: auto;
		width: calc(100% - 48px);
		max-width: 520px;
		gap: 32px;
	}
	.auth-split-layout__footer {
		margin-left: auto;
		margin-right: auto;
		width: calc(100% - 48px);
		max-width: 520px;
	}
	.auth-split-layout__slogan {
		font-size: clamp(28px, 6.5vw, 40px);
	}
	.auth-split-layout__right {
		padding: 48px 0;
	}
	.auth-split-layout__right::before {
		top: 0;
		left: 0;
		right: 0;
		bottom: auto;
		width: 100%;
		height: 1px;
		background: linear-gradient(
			to right,
			transparent 0%,
			rgba(11, 11, 13, 0.08) 20%,
			rgba(11, 11, 13, 0.08) 80%,
			transparent 100%
		);
	}
	.auth-split-layout__form-wrapper {
		margin-left: auto;
		margin-right: auto;
		width: calc(100% - 48px);
		max-width: 400px;
	}
}
`;
