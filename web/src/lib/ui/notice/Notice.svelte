<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		tone = 'neutral',
		children,
	}: { tone?: 'neutral' | 'success' | 'warning' | 'danger'; children: Snippet } = $props();

	const tones = {
		neutral: 'border-border bg-surface-raised text-fg',
		success: 'border-success/30 bg-success/10 text-fg',
		warning: 'border-warning/30 bg-warning/10 text-fg',
		danger: 'border-danger/30 bg-danger/10 text-fg',
	};
</script>

<div class="rounded-md border px-3 py-2 text-sm {tones[tone]}" role="status">
	{@render children()}
</div>
