<script lang="ts">
	// A movie or show: its poster, and its whole cast as a grid of portraits
	// that each open the actor's page, so several people from one title can
	// be fixed without searching for each.
	import { ArrowLeft } from '@lucide/svelte';
	import {
		api,
		ApiError,
		servedPortrait,
		type CastMember,
		type TitlePage,
	} from '$lib/api/client';
	import { pending } from '$lib/changes/changes.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import { link } from '$lib/router/router.svelte';

	let { ratingKey }: { ratingKey: string } = $props();

	let page = $state<TitlePage | null>(null);
	let error = $state<string | null>(null);

	function load() {
		api.title(ratingKey)
			.then((p) => (page = p))
			.catch(
				(e: ApiError) => (error = e.status === 404 ? 'No title with that key.' : e.message)
			);
	}
	$effect(() => {
		void ratingKey;
		page = null;
		error = null;
		load();
	});
	$effect(() => {
		if (pending.revision > 0) load();
	});
</script>

{#snippet member(c: CastMember)}
	<Avatar src={servedPortrait(c, 192)} alt="" size="lg" />
	<span class="block w-full truncate text-sm font-medium" title={c.name}>{c.name}</span>
	{#if c.role}
		<span class="text-fg-muted block w-full truncate text-xs" title={c.role}>{c.role}</span>
	{/if}
	{#if c.staged}
		<Badge tone="warning">staged</Badge>
	{:else if c.drift}
		<Badge tone="warning">drift</Badge>
	{:else if c.override}
		<Badge tone="success">override</Badge>
	{/if}
{/snippet}

<div class="mx-auto w-full max-w-3xl px-4 pt-4 pb-16 sm:px-6">
	{#if error}
		<p class="text-danger">{error}</p>
	{:else if !page}
		<div class="text-fg-muted flex items-center gap-3"><Spinner /> Loading from Plex…</div>
	{:else}
		<div class="flex gap-5">
			<img
				src={api.posterImage(page.title.ratingKey, 400)}
				alt=""
				class="bg-surface-raised aspect-2/3 w-28 shrink-0 rounded-lg object-cover"
			/>
			<div class="min-w-0 flex-1">
				<div class="flex items-start justify-between gap-4">
					<h1 class="text-2xl font-semibold tracking-tight">{page.title.name}</h1>
					<a
						href="/"
						use:link
						class="text-fg-muted hover:text-fg mt-2 inline-flex shrink-0 items-center gap-1.5 text-sm transition-colors"
					>
						<ArrowLeft class="size-4" aria-hidden="true" />
						Search
					</a>
				</div>
				{#if page.title.year}
					<p class="text-fg-muted mt-1 text-sm">{page.title.year}</p>
				{/if}
			</div>
		</div>

		<section class="mt-8">
			<h2 class="text-fg-muted mb-3 text-sm font-medium">Cast</h2>
			{#if page.cast.length === 0}
				<p class="text-fg-muted text-sm">Plex lists no cast for this title.</p>
			{:else}
				<ul class="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5">
					{#each page.cast as c, i (c.key + i)}
						<li>
							<a
								href="/actors/{c.key}"
								use:link
								class="hover:bg-surface-hover flex flex-col items-center gap-1.5 rounded-lg p-2 text-center transition-colors"
							>
								{@render member(c)}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>
