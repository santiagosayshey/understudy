<script lang="ts">
	import { api, type Actor, type ListingStatus } from '$lib/api/client';
	import Input from '$lib/ui/input/Input.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import Tooltip from '$lib/ui/tooltip/Tooltip.svelte';
	import { link, navigate } from '$lib/router/router.svelte';

	let { listing }: { listing: ListingStatus | null } = $props();

	let query = $state('');
	let results = $state<Actor[]>([]);
	let total = $state(0);
	let searched = $state('');
	let active = $state(-1);
	let focused = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;
	// The box only moves once typing has paused, so it does not bob while a
	// name is being typed. Clearing the box brings it straight back.
	let lifted = $state(false);
	let liftTimer: ReturnType<typeof setTimeout> | undefined;

	// How many rows fit under the search box: the viewport minus the header,
	// the box at the top, and room for the "more" line. Recomputed on resize.
	const rowHeight = 64;
	let viewport = $state(typeof window === 'undefined' ? 800 : window.innerHeight);
	const fit = $derived(
		Math.max(
			3,
			Math.floor((viewport - 56 - (lifted ? 140 : viewport * 0.3 + 90) - 48) / rowHeight)
		)
	);
	const shown = $derived(results.slice(0, fit));
	const more = $derived(total - shown.length);

	async function search() {
		const q = query.trim();
		if (!q) {
			results = [];
			total = 0;
			searched = '';
			return;
		}
		try {
			const r = await api.search(q);
			results = r.results;
			total = r.total;
			searched = q;
			active = -1;
		} catch {
			results = [];
			total = 0;
		}
	}

	function oninput() {
		clearTimeout(timer);
		timer = setTimeout(search, 120);
		clearTimeout(liftTimer);
		if (!query.trim()) lifted = false;
		else liftTimer = setTimeout(() => (lifted = query.trim() !== ''), 600);
	}

	function onkeydown(e: KeyboardEvent) {
		if (!shown.length) return;
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			active = (active + 1) % shown.length;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			active = (active - 1 + shown.length) % shown.length;
		} else if (e.key === 'Enter' && active >= 0) {
			navigate('/actors/' + shown[active].key);
		}
	}

	const ready = $derived(listing?.loaded ?? false);
</script>

<svelte:window onresize={() => (viewport = window.innerHeight)} />

<div
	class="mx-auto flex w-full max-w-xl flex-col items-center px-4 transition-[padding-top] duration-300 ease-out {lifted
		? 'pt-6'
		: 'pt-[30vh]'}"
>
	<h1
		class="overflow-hidden text-2xl font-semibold tracking-tight transition-[opacity,max-height,margin] duration-300 ease-out {lifted
			? 'mb-0 max-h-0 opacity-0'
			: 'mb-6 max-h-10 opacity-100'}"
	>
		Find an actor, choose their portrait.
	</h1>

	<div class="w-full">
		<Input
			bind:value={query}
			size="lg"
			placeholder="Search actors"
			autofocus
			{oninput}
			{onkeydown}
			onfocus={() => (focused = true)}
			onblur={() => (focused = false)}
		>
			{#snippet trailing()}
				{#if !ready && !listing?.error}
					<Tooltip text="Loading actors from Plex">
						<Spinner label="Loading actors from Plex" />
					</Tooltip>
				{:else if ready && !query}
					<Tooltip text="Ready to search">
						<svg
							viewBox="0 0 24 24"
							class="text-success size-5"
							fill="none"
							stroke="currentColor"
							stroke-width="2.5"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-label="Ready"
							role="img"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg
						>
					</Tooltip>
				{/if}
			{/snippet}
		</Input>
	</div>

	{#if !ready}
		{#if listing?.error}
			<p class="text-danger mt-4 text-sm">
				Could not load the actor listing from Plex: {listing.error}
			</p>
		{:else if focused}
			<p class="text-fg-muted mt-4 text-sm">
				Loading every actor from Plex. This takes about half a minute.
			</p>
		{/if}
	{:else if searched && results.length === 0}
		<p class="text-fg-muted mt-6 text-sm">No actor named “{searched}” in Plex.</p>
	{:else if shown.length}
		<ul
			class="divide-border border-border bg-surface shadow-raised mt-4 w-full divide-y overflow-hidden rounded-lg border"
		>
			{#each shown as actor, i (actor.key)}
				<li>
					<a
						href="/actors/{actor.key}"
						use:link
						class="hover:bg-surface-hover flex items-center gap-3 px-4 py-2.5 transition-colors {i ===
						active
							? 'bg-surface-hover'
							: ''}"
					>
						<Avatar
							src={actor.path ? api.cdnImage(actor.path, 96) : undefined}
							alt=""
							size="md"
						/>
						<span class="min-w-0 flex-1">
							<span class="block truncate font-medium">{actor.name}</span>
							<span class="text-fg-muted block truncate text-xs"
								>{actor.libraries.join(', ')}</span
							>
						</span>
						{#if actor.drift}
							<Badge tone="warning">drift</Badge>
						{:else if actor.override}
							<Badge tone="success">override</Badge>
						{/if}
						{#if !actor.path}
							<Badge>no photo</Badge>
						{/if}
					</a>
				</li>
			{/each}
		</ul>
		{#if more > 0}
			<p class="text-fg-muted mt-3 text-sm">
				{more}{total > results.length ? '+' : ''} more. Keep typing to narrow it down.
			</p>
		{/if}
	{/if}
</div>
