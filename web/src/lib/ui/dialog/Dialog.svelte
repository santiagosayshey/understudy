<script lang="ts">
	// A modal over a dimmed page. Escape and the backdrop close it. Focus
	// moves inside on open and returns on close.
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		title,
		size = 'md',
		children,
		footer,
	}: {
		open?: boolean;
		title: string;
		size?: 'md' | 'lg';
		children: Snippet;
		footer?: Snippet;
	} = $props();

	let panel = $state<HTMLElement | null>(null);
	let previous: Element | null = null;

	$effect(() => {
		if (open) {
			previous = document.activeElement;
			queueMicrotask(() =>
				panel?.querySelector<HTMLElement>('button, [href], input, [tabindex]')?.focus()
			);
			const onkey = (e: KeyboardEvent) => e.key === 'Escape' && (open = false);
			addEventListener('keydown', onkey);
			return () => {
				removeEventListener('keydown', onkey);
				(previous as HTMLElement | null)?.focus?.();
			};
		}
	});

	const sizes = { md: 'max-w-lg', lg: 'max-w-3xl' };
</script>

{#if open}
	<div class="fixed inset-0 z-40 flex items-center justify-center p-4" role="presentation">
		<button
			type="button"
			class="dialog-backdrop bg-bg/70 absolute inset-0 backdrop-blur-[2px]"
			aria-label="Close"
			onclick={() => (open = false)}
		></button>
		<div
			bind:this={panel}
			role="dialog"
			aria-modal="true"
			aria-labelledby="dialog-title"
			class="dialog-panel border-border bg-surface shadow-raised relative flex max-h-[90vh] w-full flex-col rounded-lg border {sizes[
				size
			]}"
		>
			<div class="border-border flex items-center justify-between gap-4 border-b px-5 py-3">
				<h2 id="dialog-title" class="font-medium">{title}</h2>
				<button
					type="button"
					aria-label="Close"
					onclick={() => (open = false)}
					class="text-fg-muted hover:bg-surface-hover hover:text-fg inline-flex size-8 items-center justify-center rounded-md transition-colors"
				>
					<X class="size-4" aria-hidden="true" />
				</button>
			</div>
			<div class="overflow-y-auto p-5">
				{@render children()}
			</div>
			{#if footer}
				<div class="border-border border-t px-5 py-3">
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.dialog-backdrop {
		animation: fade 150ms ease-out;
	}
	.dialog-panel {
		animation: rise 180ms cubic-bezier(0.2, 0.8, 0.2, 1);
	}
	@keyframes fade {
		from {
			opacity: 0;
		}
	}
	@keyframes rise {
		from {
			opacity: 0;
			translate: 0 8px;
			scale: 0.98;
		}
	}
</style>
