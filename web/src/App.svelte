<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Status } from '$lib/api/client';
	import { route, match } from '$lib/router/router.svelte';
	import Header from '$lib/ui/header/Header.svelte';
	import Search from './routes/Search.svelte';
	import ActorPage from './routes/ActorPage.svelte';

	let status = $state<Status | null>(null);

	// Poll status while the listing loads, then settle down.
	onMount(() => {
		let timer: ReturnType<typeof setTimeout>;
		const tick = async () => {
			try {
				status = await api.status();
			} catch {
				status = null;
			}
			timer = setTimeout(tick, status?.listing.loaded ? 30000 : 2000);
		};
		tick();
		return () => clearTimeout(timer);
	});

	const actor = $derived(match('/actors/:key', route.path));
</script>

<Header version={status?.version} />
<main>
	{#if actor}
		<ActorPage key={actor.key} />
	{:else}
		<Search listing={status?.listing ?? null} />
	{/if}
</main>
