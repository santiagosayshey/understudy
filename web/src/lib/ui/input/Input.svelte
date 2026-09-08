<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		value = $bindable(''),
		placeholder = '',
		size = 'md',
		autofocus = false,
		oninput,
		onkeydown,
		onfocus,
		onblur,
		id,
		trailing,
	}: {
		value?: string;
		placeholder?: string;
		size?: 'md' | 'lg';
		autofocus?: boolean;
		oninput?: (e: Event) => void;
		onkeydown?: (e: KeyboardEvent) => void;
		onfocus?: (e: FocusEvent) => void;
		onblur?: (e: FocusEvent) => void;
		id?: string;
		/** Something small at the right edge: a spinner, a tick, a shortcut hint. */
		trailing?: Snippet;
	} = $props();

	const sizes = { md: 'h-10 px-3 pr-10 text-sm', lg: 'h-14 px-5 pr-14 text-lg' };
</script>

<div class="relative w-full">
	<!-- svelte-ignore a11y_autofocus -->
	<input
		{id}
		type="search"
		bind:value
		{placeholder}
		{autofocus}
		{oninput}
		{onkeydown}
		{onfocus}
		{onblur}
		autocomplete="off"
		spellcheck="false"
		class="border-border bg-surface text-fg shadow-raised placeholder:text-fg-muted hover:border-border-strong focus:border-fg-muted w-full rounded-lg border transition-colors focus:outline-none {sizes[
			size
		]}"
	/>
	{#if trailing}
		<span
			class="pointer-events-none absolute inset-y-0 right-0 flex items-center {size === 'lg'
				? 'pr-5'
				: 'pr-3'}"
		>
			{@render trailing()}
		</span>
	{/if}
</div>
