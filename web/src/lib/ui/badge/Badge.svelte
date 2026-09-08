<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		tone = 'neutral',
		children,
	}: { tone?: 'neutral' | 'success' | 'warning' | 'danger'; children: Snippet } = $props();

	const tones = {
		neutral: 'border-border bg-surface-raised text-fg-muted',
		success: 'border-success/30 bg-success/10 text-success',
		warning: 'border-warning/30 bg-warning/10 text-warning',
		danger: 'border-danger/30 bg-danger/10 text-danger',
	};
</script>

<span
	class="inline-flex items-center rounded-sm border px-2 py-0.5 text-xs font-medium whitespace-nowrap {tones[
		tone
	]}">{@render children()}</span
>
