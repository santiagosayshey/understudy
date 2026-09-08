<script lang="ts">
	// The classic mode toggle: the sun turns and shrinks away while the moon
	// turns and grows in, and back.
	import { Sun, Moon } from '@lucide/svelte';
	import { theme, setTheme } from '$lib/theme/theme.svelte';

	const dark = $derived(theme.value === 'dark');
	function toggle() {
		setTheme(dark ? 'light' : 'dark');
	}
</script>

<button
	type="button"
	role="switch"
	aria-checked={dark}
	aria-label="Dark mode"
	onclick={toggle}
	class="text-fg border-border bg-surface shadow-raised hover:border-border-strong hover:bg-surface-hover focus-visible:outline-ring relative inline-flex size-9 items-center justify-center rounded-md border transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
>
	<Sun
		class="absolute size-5 transition-all duration-300 {dark
			? 'scale-0 -rotate-90'
			: 'scale-100 rotate-0'}"
		aria-hidden="true"
	/>
	<Moon
		class="absolute size-5 transition-all duration-300 {dark
			? 'scale-100 rotate-0'
			: 'scale-0 rotate-90'}"
		aria-hidden="true"
	/>
</button>
