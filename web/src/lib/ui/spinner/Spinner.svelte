<script lang="ts">
	let { size = 'md', label = 'Loading' }: { size?: 'sm' | 'md'; label?: string } = $props();
	const sizes = { sm: 'size-4 border-2', md: 'size-5 border-2' };
</script>

<span
	class="text-fg-muted inline-block animate-spin rounded-full border-current border-t-transparent {sizes[
		size
	]}"
	role="status"
	aria-label={label}
></span>
