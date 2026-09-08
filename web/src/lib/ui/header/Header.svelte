<script lang="ts">
	import ThemeToggle from '$lib/ui/theme-toggle/ThemeToggle.svelte';
	import { link } from '$lib/router/router.svelte';

	let { version = '' }: { version?: string } = $props();
</script>

<header class="flex h-14 items-center justify-between px-4 sm:px-6">
	<a
		href="/"
		use:link
		class="text-fg hover:bg-surface-hover focus-visible:outline-ring -mx-2 flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
	>
		<img src="/icon.png" alt="" class="size-6" />
		<span class="font-semibold tracking-tight">understudy</span>
		{#if version}
			<span class="text-fg-muted font-mono text-xs">/{version}</span>
		{/if}
	</a>
	<ThemeToggle />
</header>
