<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		variant = 'primary',
		href,
		type = 'button',
		disabled = false,
		onclick,
		children,
	}: {
		variant?: 'primary' | 'secondary' | 'danger';
		href?: string;
		type?: 'button' | 'submit';
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
	} = $props();

	const base =
		'inline-flex items-center gap-2 rounded-[var(--radius)] px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:pointer-events-none disabled:opacity-50';
	const variants = {
		primary: 'bg-accent text-accent-foreground hover:brightness-110',
		secondary: 'border border-border bg-surface text-foreground hover:border-muted',
		danger: 'border border-danger text-danger hover:bg-danger/10',
	};
</script>

{#if href}
	<a {href} class="{base} {variants[variant]}">{@render children()}</a>
{:else}
	<button {type} {disabled} {onclick} class="{base} {variants[variant]}"
		>{@render children()}</button
	>
{/if}
