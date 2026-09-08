<script lang="ts">
	// The round portrait, the shape Plex draws in a cast list. No src means
	// Plex has no photo for the person, so a placeholder stands in.
	let {
		src,
		alt = '',
		size = 'md',
	}: { src?: string; alt?: string; size?: 'sm' | 'md' | 'lg' | 'xl' } = $props();

	const sizes = { sm: 'size-8', md: 'size-12', lg: 'size-24', xl: 'size-40' };
</script>

<span
	class="bg-surface-raised text-fg-muted inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full {sizes[
		size
	]}"
>
	{#if src}
		<img {src} {alt} class="size-full object-cover" loading="lazy" />
	{:else}
		<svg viewBox="0 0 24 24" class="size-1/2" fill="currentColor" aria-hidden="true">
			<path
				d="M12 12a5 5 0 1 0 0-10 5 5 0 0 0 0 10Zm0 2c-4.42 0-8 2.24-8 5v1h16v-1c0-2.76-3.58-5-8-5Z"
			/>
		</svg>
	{/if}
</span>
