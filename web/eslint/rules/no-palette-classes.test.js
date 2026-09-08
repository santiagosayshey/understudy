import { test } from 'node:test';
import { RuleTester } from 'eslint';
import svelteParser from 'svelte-eslint-parser';
import rule from './no-palette-classes.js';

const tester = new RuleTester({ languageOptions: { parser: svelteParser } });

test('no-palette-classes', () => {
	tester.run('no-palette-classes', rule, {
		valid: [
			'<div class="bg-surface text-fg-muted border-border hover:bg-surface-hover">ok</div>',
			'<div class="text-[0.6875rem] size-5 rounded-md shadow-raised">sizes are fine</div>',
			'<div class="border-current border-t-transparent">current and transparent are fine</div>',
			'<div class={dark ? "bg-accent" : "bg-surface"}>ok</div>',
		],
		invalid: [
			{
				code: '<div class="bg-neutral-200">no</div>',
				errors: [{ messageId: 'palette', data: { found: 'bg-neutral-200' } }],
			},
			{ code: '<div class="text-red-500/50">no</div>', errors: [{ messageId: 'palette' }] },
			{ code: '<div class="hover:bg-zinc-800">no</div>', errors: [{ messageId: 'palette' }] },
			{ code: '<div class="bg-white">no</div>', errors: [{ messageId: 'palette' }] },
			{ code: '<div class="bg-[#fff]">no</div>', errors: [{ messageId: 'palette' }] },
			{
				code: '<div class={dark ? "bg-neutral-900" : "bg-surface"}>no</div>',
				errors: [{ messageId: 'palette' }],
			},
			{
				code: '<div class="x {cond ? `bg-red-500` : ``}">no</div>',
				errors: [{ messageId: 'palette' }],
			},
			{ code: '<div style="color: #333">no</div>', errors: [{ messageId: 'style' }] },
			{
				code: '<div class="scrollbar-thumb-neutral-400">no</div>',
				errors: [{ messageId: 'palette' }],
			},
		],
	});
});
