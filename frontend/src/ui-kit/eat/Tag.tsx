import { Tag as AntTag } from "antd";
import type { TagProps } from "antd";

// Tag wraps antd Tag with bordered=false by default so all tags in the app
// render without a border. Pass bordered={true} explicitly to override.
export function Tag({ bordered = true, ...props }: TagProps) {
	return <AntTag bordered={bordered} {...props} />;
}
