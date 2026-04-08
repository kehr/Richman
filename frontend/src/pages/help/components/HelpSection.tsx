import type { HelpBlock, HelpSection as HelpSectionModel } from "@/i18n/help";
import { Alert, Table, Typography } from "@/ui-kit/eat";

const { Title, Paragraph } = Typography;

// HelpSection renders a single chapter of the help document. The section id
// is applied to the heading element so `/help#badge` anchors work, and the
// block list is rendered via a plain switch on the block `type` discriminator.
// Keep this renderer in sync with the block union defined in
// `frontend/src/i18n/help/types.ts` — adding a new block type there requires
// a matching case here.

interface HelpSectionProps {
	section: HelpSectionModel;
}

interface BlockRendererProps {
	block: HelpBlock;
}

function BlockRenderer({ block }: BlockRendererProps) {
	switch (block.type) {
		case "paragraph":
			return <Paragraph style={{ marginBottom: 12 }}>{block.text}</Paragraph>;
		case "list": {
			const ListTag = block.ordered ? "ol" : "ul";
			return (
				<ListTag style={{ marginBottom: 16, paddingLeft: 20 }}>
					{block.items.map((item, itemIndex) => (
						// Help content is static JSON and list items never reorder at
						// runtime, so the authored index is a stable React key and is
						// safer than keying on the item text which may repeat.
						// biome-ignore lint/suspicious/noArrayIndexKey: static authored content
						<li key={itemIndex} style={{ marginBottom: 4 }}>
							{item}
						</li>
					))}
				</ListTag>
			);
		}
		case "table": {
			// Antd Table needs column defs; build them from the header row so
			// authors only have to maintain one list of headers per block.
			const columns = block.headers.map((header, i) => ({
				title: header,
				dataIndex: `col${i}`,
				key: `col${i}`,
			}));
			const dataSource = block.rows.map((row, rowIndex) => {
				const record: Record<string, string> = { key: String(rowIndex) };
				row.forEach((cell, i) => {
					record[`col${i}`] = cell;
				});
				return record;
			});
			return (
				<Table
					size="small"
					bordered
					pagination={false}
					columns={columns}
					dataSource={dataSource}
					style={{ marginBottom: 16 }}
				/>
			);
		}
		case "code":
			return (
				<pre
					style={{
						background: "#f5f5f5",
						border: "1px solid #e8e8e8",
						borderRadius: 4,
						padding: 12,
						marginBottom: 16,
						overflowX: "auto",
					}}
				>
					<code>{block.code}</code>
				</pre>
			);
		case "note":
			return <Alert type="info" showIcon message={block.text} style={{ marginBottom: 16 }} />;
		default:
			return null;
	}
}

export function HelpSection({ section }: HelpSectionProps) {
	return (
		<section
			id={section.id}
			data-testid={`help-section-${section.id}`}
			style={{ marginBottom: 48, scrollMarginTop: 80 }}
		>
			<Title level={2} style={{ marginTop: 0 }}>
				{section.title}
			</Title>
			{section.blocks.map((block, index) => (
				// Help content is authored as a static JSON document; blocks never
				// reorder or get inserted at runtime, so a type+index composite is
				// stable across renders without hashing every block body.
				<BlockRenderer key={`${block.type}-${index}`} block={block} />
			))}
		</section>
	);
}
