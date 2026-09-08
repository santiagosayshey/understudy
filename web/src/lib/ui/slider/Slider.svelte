<script lang="ts">
	let {
		value = $bindable(0),
		min = 0,
		max = 1,
		step = 0.01,
		label,
		oninput,
	}: {
		value?: number;
		min?: number;
		max?: number;
		step?: number;
		label: string;
		oninput?: (e: Event) => void;
	} = $props();
</script>

<input
	type="range"
	bind:value
	{min}
	{max}
	{step}
	{oninput}
	aria-label={label}
	class="slider bg-border accent-accent focus-visible:outline-ring h-1.5 w-full cursor-pointer appearance-none rounded-full focus-visible:outline-2 focus-visible:outline-offset-4"
/>
