<script lang="ts">
	// A file picker with any look: the input is hidden inside a label and the
	// label gets the classes, so a dashed drop frame and a plain text button
	// are the same component.
	import type { Snippet } from 'svelte';

	let {
		accept = 'image/*',
		class: cls = '',
		onfile,
		children,
	}: {
		accept?: string;
		class?: string;
		onfile: (file: File) => void;
		children: Snippet;
	} = $props();

	function onchange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (file) onfile(file);
		input.value = '';
	}
</script>

<label
	class="has-focus-visible:outline-ring cursor-pointer has-focus-visible:outline-2 has-focus-visible:outline-offset-2 {cls}"
>
	<input type="file" {accept} class="sr-only" {onchange} />
	{@render children()}
</label>
