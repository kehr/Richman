import md5 from "blueimp-md5";

// gravatarUrl returns the Gravatar avatar URL for the given email.
// An empty email returns "" — antd Avatar treats empty src as load failure
// and automatically falls back to the icon prop.
export function gravatarUrl(email: string, size = 32): string {
	if (!email) return "";
	const hash = md5(email.trim().toLowerCase());
	return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=${size}&r=g`;
}
