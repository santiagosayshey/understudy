<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		class: cls = '',
		children,
	}: { title?: string; class?: string; children: Snippet } = $props();
</script>

<section class="border-border bg-surface shadow-raised rounded-lg border p-5 {cls}">
	{#if title}
		<h2 class="text-fg-muted mb-4 text-sm font-medium">{title}</h2>
	{/if}
	{@render children()}
</section>
