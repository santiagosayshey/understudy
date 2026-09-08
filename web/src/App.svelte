<script lang="ts">
	import Button from '$lib/ui/button/Button.svelte';

	type Status = { version: string };
	let status = $state<Status | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		fetch('/api/status')
			.then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.statusText))))
			.then((s: Status) => (status = s))
			.catch((e: Error) => (error = e.message));
	});
</script>

<main class="flex min-h-screen flex-col items-center justify-center gap-6 p-8 text-center">
	<img src="/icon.png" alt="" class="size-24" />
	<h1 class="text-4xl font-semibold tracking-tight">Understudy</h1>
	<p class="text-muted max-w-md">
		Your own actor portraits for Plex. Understudy stands in for Plex's image CDN, serves the
		photos you chose, and keeps them attached to the right people.
	</p>
	<p class="text-muted text-sm">
		{#if status}
			version {status.version}
		{:else if error}
			the API is not reachable: {error}
		{:else}
			checking the API…
		{/if}
	</p>
	<Button
		href="https://github.com/santiagosayshey/understudy/blob/develop/docs/design.md"
		variant="secondary">Design document</Button
	>
</main>
