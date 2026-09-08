// Forbids Tailwind palette steps and literal colours in markup. Components
// read colour only through the semantic tokens in app.css, so a theme is a
// change of token values and nothing else.
const palettes =
	'slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|black|white';
const prefixes =
	'bg|text|border|ring|outline|fill|stroke|from|to|via|divide|placeholder|shadow|accent|caret|decoration';
const palette = new RegExp(
	`(?:^|[\\s:!])(?:${prefixes})-(?:${palettes})(?:-\\d{2,3})?(?:/\\d{1,3})?(?=$|[\\s"'])`
);
const literal = new RegExp(
	`(?:${prefixes})-\\[(?:#[0-9a-fA-F]{3,8}|rgba?\\(|hsla?\\(|oklch\\(|oklab\\()`
);
const styleColour =
	/(?:^|;)\s*(?:background(?:-color)?|color|border(?:-color)?|fill|stroke)\s*:\s*(?:#|rgba?\(|hsla?\(|oklch\(|oklab\()/;

function offending(text) {
	const m = text.match(palette) ?? text.match(literal);
	return m ? m[0].trim().replace(/^[:!]/, '') : null;
}

export default {
	meta: {
		type: 'suggestion',
		docs: { description: 'use semantic colour tokens, not palette classes' },
		schema: [],
		messages: {
			palette:
				'Use a semantic token (bg-surface, text-fg-muted, border-border, …) instead of "{{found}}".',
			style: 'Set colours through the semantic tokens, not an inline style.',
		},
	},
	create(context) {
		function checkText(node, text) {
			const found = offending(text);
			if (found) context.report({ node, messageId: 'palette', data: { found } });
		}
		function checkValue(node) {
			if (node.type === 'SvelteLiteral') checkText(node, node.value);
			else if (node.type === 'SvelteMustacheTag') walk(node.expression);
		}
		function walk(expr) {
			if (!expr || typeof expr !== 'object') return;
			switch (expr.type) {
				case 'Literal':
					if (typeof expr.value === 'string') checkText(expr, expr.value);
					break;
				case 'TemplateLiteral':
					for (const q of expr.quasis) checkText(q, q.value.cooked ?? '');
					for (const e of expr.expressions) walk(e);
					break;
				case 'ConditionalExpression':
					walk(expr.consequent);
					walk(expr.alternate);
					break;
				case 'LogicalExpression':
					walk(expr.left);
					walk(expr.right);
					break;
				case 'ArrayExpression':
					for (const e of expr.elements) walk(e);
					break;
				case 'ObjectExpression':
					for (const p of expr.properties) {
						if (p.type === 'Property') {
							if (p.key.type === 'Literal') checkText(p.key, String(p.key.value));
							if (p.key.type === 'Identifier') checkText(p.key, p.key.name);
							walk(p.value);
						}
					}
					break;
			}
		}
		return {
			SvelteAttribute(node) {
				const name = node.key.name;
				if (name === 'class') {
					for (const v of node.value) checkValue(v);
				} else if (name === 'style') {
					for (const v of node.value) {
						if (v.type === 'SvelteLiteral' && styleColour.test(v.value)) {
							context.report({ node: v, messageId: 'style' });
						}
					}
				}
			},
			// class:foo={cond} and the script side: string literals assigned to
			// anything named like a class list
			VariableDeclarator(node) {
				if (!node.id?.name || !/class|variant|tone|size|styles?$/i.test(node.id.name))
					return;
				walk(node.init);
			},
		};
	},
};
