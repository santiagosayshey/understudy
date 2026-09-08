<script lang="ts">
	// A button that opens the file picker. The input is hidden inside the
	// label so the styling is the Button's.
	import type { Snippet } from 'svelte';

	let {
		accept = 'image/*',
		variant = 'primary',
		onfile,
		children,
	}: {
		accept?: string;
		variant?: 'primary' | 'secondary';
		onfile: (file: File) => void;
		children: Snippet;
	} = $props();

	const base =
		'inline-flex h-10 cursor-pointer items-center justify-center gap-2 rounded-md px-4 text-sm font-medium whitespace-nowrap transition-colors select-none has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-ring';
	const variants = {
		primary: 'bg-accent text-accent-fg hover:bg-accent-hover',
		secondary:
			'border border-border bg-surface text-fg hover:border-border-strong hover:bg-surface-hover',
	};

	function onchange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (file) onfile(file);
		input.value = '';
	}
</script>

<label class="{base} {variants[variant]}">
	<input type="file" {accept} class="sr-only" {onchange} />
	{@render children()}
</label>
