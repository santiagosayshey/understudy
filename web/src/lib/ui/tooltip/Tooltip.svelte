<script lang="ts">
	// A small label that appears above its trigger on hover or focus,
	// instantly, with a short slide in. Announced through aria-describedby.
	import type { Snippet } from 'svelte';

	let {
		text,
		side = 'top',
		children,
	}: { text: string; side?: 'top' | 'bottom'; children: Snippet } = $props();

	const id = 'tip-' + Math.random().toString(36).slice(2, 8);
	let open = $state(false);

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') open = false;
	}
</script>

<span
	class="pointer-events-auto relative inline-flex"
	aria-describedby={open ? id : undefined}
	onmouseenter={() => (open = true)}
	onmouseleave={() => (open = false)}
	onfocusin={() => (open = true)}
	onfocusout={() => (open = false)}
	{onkeydown}
	role="presentation"
>
	{@render children()}
	{#if open}
		<span
			{id}
			role="tooltip"
			class="tooltip bg-accent text-accent-fg shadow-raised pointer-events-none absolute left-1/2 z-20 w-max max-w-xs -translate-x-1/2 rounded-md px-2 py-1 text-[0.6875rem] leading-tight font-medium {side ===
			'top'
				? 'bottom-full mb-1.5'
				: 'top-full mt-1.5'}"
			data-side={side}
		>
			{text}
		</span>
	{/if}
</span>

<style>
	/* Centring and the entrance both use the translate property, so they
	   never fight: Tailwind's translate utilities are deliberately not used. */
	.tooltip {
		translate: -50% 0;
		animation: tooltip-in 120ms ease-out;
	}
	@keyframes tooltip-in {
		from {
			opacity: 0;
			translate: -50% var(--tooltip-from, 4px);
		}
		to {
			opacity: 1;
			translate: -50% 0;
		}
	}
	.tooltip[data-side='bottom'] {
		--tooltip-from: -4px;
	}
</style>
