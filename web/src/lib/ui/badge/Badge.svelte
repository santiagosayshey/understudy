<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ExternalLink } from '@lucide/svelte';

	// With href the badge is a link to another site, opened in a new tab.
	let {
		tone = 'neutral',
		href,
		children,
	}: {
		tone?: 'neutral' | 'success' | 'warning' | 'danger';
		href?: string;
		children: Snippet;
	} = $props();

	const tones = {
		neutral: 'border-border bg-surface-raised text-fg-muted',
		success: 'border-success/30 bg-success/10 text-success',
		warning: 'border-warning/30 bg-warning/10 text-warning',
		danger: 'border-danger/30 bg-danger/10 text-danger',
	};
	const base =
		'inline-flex items-center rounded-sm border px-2 py-0.5 text-xs font-medium whitespace-nowrap';
</script>

{#if href}
	<a
		{href}
		target="_blank"
		rel="noreferrer"
		class="{base} hover:border-border-strong hover:text-fg focus-visible:outline-ring gap-1 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 {tones[
			tone
		]}"
	>
		{@render children()}
		<ExternalLink class="size-3" aria-hidden="true" />
	</a>
{:else}
	<span class="{base} {tones[tone]}">{@render children()}</span>
{/if}
