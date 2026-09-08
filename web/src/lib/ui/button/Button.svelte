<script lang="ts">
	import type { Snippet } from 'svelte';
	import { link } from '$lib/router/router.svelte';

	let {
		variant = 'primary',
		size = 'md',
		href,
		type = 'button',
		disabled = false,
		loading = false,
		onclick,
		children,
	}: {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		href?: string;
		type?: 'button' | 'submit';
		disabled?: boolean;
		loading?: boolean;
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
	} = $props();

	const base =
		'inline-flex items-center justify-center gap-2 rounded-md font-medium whitespace-nowrap transition-colors select-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50';
	const sizes = { sm: 'h-8 px-3 text-sm', md: 'h-10 px-4 text-sm', lg: 'h-12 px-6 text-base' };
	const variants = {
		primary: 'bg-accent text-accent-fg hover:bg-accent-hover',
		secondary:
			'border border-border bg-surface text-fg hover:border-border-strong hover:bg-surface-hover',
		ghost: 'text-fg hover:bg-surface-hover',
		danger: 'border border-danger/40 text-danger hover:bg-danger/10',
	};
	const cls = $derived(`${base} ${sizes[size]} ${variants[variant]}`);
</script>

{#if href}
	<a {href} class={cls} use:link>{@render children()}</a>
{:else}
	<button {type} disabled={disabled || loading} {onclick} class={cls} aria-busy={loading}>
		{#if loading}
			<span
				class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
				aria-hidden="true"
			></span>
		{/if}
		{@render children()}
	</button>
{/if}
