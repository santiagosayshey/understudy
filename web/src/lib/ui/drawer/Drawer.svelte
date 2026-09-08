<script lang="ts">
	// A panel that slides in from the right over a dimmed page, with a
	// footer for the actions. Escape and the backdrop close it.
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		title,
		children,
		footer,
	}: { open?: boolean; title: string; children: Snippet; footer?: Snippet } = $props();

	$effect(() => {
		if (open) {
			const onkey = (e: KeyboardEvent) => e.key === 'Escape' && (open = false);
			addEventListener('keydown', onkey);
			return () => removeEventListener('keydown', onkey);
		}
	});
</script>

{#if open}
	<div class="fixed inset-0 z-40" role="presentation">
		<button
			type="button"
			class="drawer-backdrop bg-bg/70 absolute inset-0 backdrop-blur-[2px]"
			aria-label="Close"
			onclick={() => (open = false)}
		></button>
		<div
			role="dialog"
			aria-modal="true"
			aria-labelledby="drawer-title"
			class="drawer-panel border-border bg-surface shadow-raised absolute inset-y-0 right-0 flex w-full max-w-md flex-col border-l"
		>
			<div class="border-border flex items-center justify-between gap-4 border-b px-5 py-3">
				<h2 id="drawer-title" class="font-medium">{title}</h2>
				<button
					type="button"
					aria-label="Close"
					onclick={() => (open = false)}
					class="text-fg-muted hover:bg-surface-hover hover:text-fg inline-flex size-8 items-center justify-center rounded-md transition-colors"
				>
					<X class="size-4" aria-hidden="true" />
				</button>
			</div>
			<div class="flex-1 overflow-y-auto p-5">
				{@render children()}
			</div>
			{#if footer}
				<div class="border-border border-t px-5 py-4">
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.drawer-backdrop {
		animation: fade 150ms ease-out;
	}
	.drawer-panel {
		animation: slide 220ms cubic-bezier(0.2, 0.8, 0.2, 1);
	}
	@keyframes fade {
		from {
			opacity: 0;
		}
	}
	@keyframes slide {
		from {
			translate: 100% 0;
		}
	}
</style>
