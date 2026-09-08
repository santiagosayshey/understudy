<script lang="ts">
	// A picture that is a button: one of TMDb's portraits in the picker,
	// poster-shaped like the titles beside it, a click away from the crop.
	let {
		src,
		label,
		onclick,
		class: cls = '',
	}: { src: string; label: string; onclick: () => void; class?: string } = $props();
</script>

<button
	type="button"
	aria-label={label}
	{onclick}
	class="bg-surface-raised focus-visible:outline-ring block overflow-hidden rounded-md transition-opacity hover:opacity-80 focus-visible:outline-2 focus-visible:outline-offset-2 {cls}"
>
	<img {src} alt="" class="aspect-2/3 w-full object-cover" loading="lazy" />
</button>
