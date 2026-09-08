import { test } from 'node:test';
import { RuleTester } from 'eslint';
import svelteParser from 'svelte-eslint-parser';
import rule from './no-raw-elements.js';

const tester = new RuleTester({ languageOptions: { parser: svelteParser } });

test('no-raw-elements', () => {
	tester.run('no-raw-elements', rule, {
		valid: [
			{ code: '<Button>ok</Button>', filename: '/src/routes/Search.svelte' },
			{ code: '<a href="/">ok</a>', filename: '/src/routes/Search.svelte' },
			{ code: '<button>ok</button>', filename: '/src/lib/ui/button/Button.svelte' },
		],
		invalid: [
			{
				code: '<button onclick={f}>no</button>',
				filename: '/src/routes/Search.svelte',
				errors: [
					{
						messageId: 'raw',
						data: { element: 'button', component: '<Button> from $lib/ui/button' },
					},
				],
			},
			{
				code: '<input type="text" />',
				filename: '/src/routes/Actor.svelte',
				errors: [{ messageId: 'raw' }],
			},
		],
	});
});
