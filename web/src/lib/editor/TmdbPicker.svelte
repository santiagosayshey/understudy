<script lang="ts">
	// The person elsewhere: links to them on TMDb and IMDb, and, when the
	// server has a TMDb key, the portraits TMDb holds, each a click from the
	// crop editor. The server picks the likeliest person of that name; the
	// others are a click away when it picked wrong.
	import { ExternalLink } from '@lucide/svelte';
	import { api, type TmdbMatch, type TmdbPerson } from '$lib/api/client';
	import Button from '$lib/ui/button/Button.svelte';
	import ImageButton from '$lib/ui/image-button/ImageButton.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';

	let {
		actorKey,
		actorName,
		enabled,
		onpick,
	}: {
		actorKey: string;
		actorName: string;
		/** whether the server has a key; without one there are only links */
		enabled: boolean;
		onpick: (path: string) => void;
	} = $props();

	let match = $state<TmdbMatch | null>(null);
	let person = $state<TmdbPerson | null>(null);
	let error = $state<string | null>(null);
	let loading = $state(false);

	$effect(() => {
		void actorKey;
		match = null;
		person = null;
		error = null;
		if (!enabled) return;
		loading = true;
		api.tmdb(actorKey)
			.then((m) => {
				match = m;
				person = m.person;
			})
			.catch((e: Error) => (error = e.message))
			.finally(() => (loading = false));
	});

	async function choose(id: number) {
		error = null;
		loading = true;
		try {
			person = await api.tmdbPerson(id);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	const query = $derived(encodeURIComponent(actorName));
	const tmdbHref = $derived(
		person
			? `https://www.themoviedb.org/person/${person.id}`
			: `https://www.themoviedb.org/search?query=${query}`
	);
	const imdbHref = $derived(
		person?.imdbId
			? `https://www.imdb.com/name/${person.imdbId}/`
			: `https://www.imdb.com/find/?q=${query}`
	);
	const others = $derived((match?.candidates ?? []).filter((c) => c.id !== person?.id));
	const linkClass =
		'text-fg-muted hover:text-fg inline-flex items-center gap-1 text-sm transition-colors';
</script>

<section class="mt-8">
	<div class="mb-3 flex flex-wrap items-center gap-x-4 gap-y-1">
		<h2 class="text-fg-muted text-sm font-medium">Elsewhere</h2>
		<a href={tmdbHref} target="_blank" rel="noreferrer" class={linkClass}>
			TMDb <ExternalLink class="size-3.5" aria-hidden="true" />
		</a>
		<a href={imdbHref} target="_blank" rel="noreferrer" class={linkClass}>
			IMDb <ExternalLink class="size-3.5" aria-hidden="true" />
		</a>
	</div>
	{#if !enabled}
		<p class="text-fg-muted text-sm">
			Set UNDERSTUDY_TMDB_KEY and the portraits TMDb has appear here.
		</p>
	{:else if error}
		<Notice tone="danger">{error}</Notice>
	{:else if loading}
		<div class="text-fg-muted flex items-center gap-3 text-sm"><Spinner /> Asking TMDb…</div>
	{:else if !person}
		<p class="text-fg-muted text-sm">No one on TMDb by that name.</p>
	{:else}
		{#if person.profiles.length}
			<ul class="flex gap-4 overflow-x-auto pb-4">
				{#each person.profiles as p (p.path)}
					<li class="w-28 shrink-0">
						<ImageButton
							src={api.tmdbImage(p.path, 400)}
							label="Use this portrait of {person.name}"
							onclick={() => onpick(p.path)}
						/>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="text-fg-muted text-sm">TMDb has no pictures of {person.name}.</p>
		{/if}
		{#if others.length}
			<div class="mt-1 flex flex-wrap items-center gap-2">
				<span class="text-fg-muted text-sm">Not {person.name}?</span>
				{#each others as c (c.id)}
					<Button variant="secondary" size="sm" onclick={() => choose(c.id)}>
						{c.name}{c.knownFor.length ? ` · ${c.knownFor[0]}` : ''}
					</Button>
				{/each}
			</div>
		{/if}
	{/if}
</section>
