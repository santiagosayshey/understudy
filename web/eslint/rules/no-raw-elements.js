// Forbids the raw HTML elements the ui library wraps. Only the library itself
// may use them, so every button and input on a screen is the same button
// and input.
const components = {
	button: '<Button> from $lib/ui/button',
	input: '<Input> from $lib/ui/input',
	select: 'a select component in $lib/ui',
	textarea: 'a textarea component in $lib/ui',
};

export default {
	meta: {
		type: 'suggestion',
		docs: { description: 'use the ui library instead of raw form elements' },
		schema: [
			{
				type: 'object',
				properties: { allow: { type: 'array', items: { type: 'string' } } },
				additionalProperties: false,
			},
		],
		messages: { raw: 'Use {{component}} instead of a raw <{{element}}>.' },
	},
	create(context) {
		const allow = new RegExp((context.options[0]?.allow ?? ['/src/lib/ui/']).join('|'));
		if (allow.test(context.filename.replaceAll('\\', '/'))) return {};
		return {
			SvelteElement(node) {
				if (node.kind !== 'html') return;
				const element = node.name.name;
				if (!(element in components)) return;
				context.report({
					node: node.startTag,
					messageId: 'raw',
					data: { element, component: components[element] },
				});
			},
		};
	},
};
