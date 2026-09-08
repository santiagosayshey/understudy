<script lang="ts">
	import { ArrowLeft, Check, Copy, ImagePlus, Info } from '@lucide/svelte';
	import { api, ApiError, type ActorPage, type UploadSource } from '$lib/api/client';
	import { discard, stage, pending } from '$lib/changes/changes.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Dialog from '$lib/ui/dialog/Dialog.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import Tooltip from '$lib/ui/tooltip/Tooltip.svelte';
	import { link } from '$lib/router/router.svelte';
	import PortraitDialog from '$lib/editor/PortraitDialog.svelte';
	import { TmdbLookup } from '$lib/editor/tmdb.svelte';

	let { key, tmdb }: { key: string; tmdb: boolean } = $props();

	let page = $state<ActorPage | null>(null);
	let error = $state<string | null>(null);
	let source = $state<UploadSource | null>(null);
	// the actor on TMDb, looked up quietly once the page has them
	const lookup = new TmdbLookup();
	let editing = $state(false);
	let over = $state(false);
	let details = $state(false);
	let copied = $state('');

	async function copy(value: string) {
		await navigator.clipboard.writeText(value);
		copied = value;
		setTimeout(() => (copied = ''), 1200);
	}

	function load() {
		api.actor(key)
			.then((p) => (page = p))
			.catch(
				(e: ApiError) => (error = e.status === 404 ? 'No actor with that key.' : e.message)
			);
	}
	$effect(() => {
		void key;
		page = null;
		error = null;
		load();
	});
	// after any stage, discard or apply, reload in place so the page shows
	// the current state, including changes made from the drawer
	$effect(() => {
		if (pending.revision > 0) load();
	});
	$effect(() => {
		if (tmdb) lookup.load(key);
	});

	const actor = $derived(page?.actor);
	const override = $derived(page?.override);
	const staged = $derived(page?.staged);
	// what Plex will serve once everything is applied
	const effective = $derived.by(() => {
		if (!actor) return undefined;
		if (staged?.kind === 'set') return api.changeImage(actor.key, staged.stagedAt);
		if (staged?.kind === 'remove')
			return actor.path ? api.cdnImage(actor.path, 400) : undefined;
		if (override) return api.portraitImage(override.image);
		return actor.path ? api.cdnImage(actor.path, 400) : undefined;
	});
	const showsPlex = $derived(
		!!actor?.path && (!!override || staged?.kind === 'set') && staged?.kind !== 'remove'
	);
	const libraries = $derived.by(() => {
		const groups: [string, NonNullable<typeof actor>['titles']][] = [];
		for (const t of actor?.titles ?? []) {
			let g = groups.find(([name]) => name === t.library);
			if (!g) {
				g = [t.library, []];
				groups.push(g);
			}
			g[1].push(t);
		}
		return groups;
	});

	// the person on TMDb and IMDb: their own pages once TMDb found them, a
	// search for the name until then
	const tmdbHref = $derived(
		lookup.person
			? `https://www.themoviedb.org/person/${lookup.person.id}`
			: `https://www.themoviedb.org/search?query=${encodeURIComponent(actor?.name ?? '')}`
	);
	const imdbHref = $derived(
		lookup.person?.imdbId
			? `https://www.imdb.com/name/${lookup.person.imdbId}/`
			: `https://www.imdb.com/find/?q=${encodeURIComponent(actor?.name ?? '')}`
	);

	// a file dropped on the portrait skips the picker
	function choose(f: File) {
		source = { file: f };
		editing = true;
	}
	function ondrop(e: DragEvent) {
		e.preventDefault();
		over = false;
		const f = e.dataTransfer?.files?.[0];
		if (f && f.type.startsWith('image/')) choose(f);
	}
	async function remove() {
		if (!actor) return;
		await stage({ key: actor.key, kind: 'remove' });
	}
	async function undo() {
		if (!actor) return;
		await discard(actor.key);
	}
</script>

<div class="mx-auto w-full max-w-3xl px-4 pt-4 pb-16 sm:px-6">
	{#if error}
		<p class="text-danger">{error}</p>
	{:else if !actor}
		<div class="text-fg-muted flex items-center gap-3"><Spinner /> Loading from Plex…</div>
	{:else}
		<div class="flex flex-col gap-6 sm:flex-row sm:items-start">
			<div
				class="flex shrink-0 flex-col items-center gap-3 rounded-full transition-shadow {over
					? 'ring-ring/40 ring-4'
					: ''}"
				role="presentation"
				ondragover={(e) => {
					e.preventDefault();
					over = true;
				}}
				ondragleave={() => (over = false)}
				{ondrop}
			>
				<Avatar src={effective} alt="" size="xl" />
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-start justify-between gap-4">
					<div class="flex items-center gap-1">
						<h1 class="text-3xl font-semibold tracking-tight">{actor.name}</h1>
						<Button
							variant="ghost"
							size="icon"
							label="Details"
							onclick={() => (details = true)}
						>
							<Info class="size-4" aria-hidden="true" />
						</Button>
					</div>
					<a
						href="/"
						use:link
						class="text-fg-muted hover:text-fg mt-2 inline-flex shrink-0 items-center gap-1.5 text-sm transition-colors"
					>
						<ArrowLeft class="size-4" aria-hidden="true" />
						Search
					</a>
				</div>
				<div class="mt-3 flex flex-wrap gap-2">
					<Badge href={tmdbHref}>TMDb</Badge>
					<Badge href={imdbHref}>IMDb</Badge>
					{#if staged}
						<Badge tone="warning"
							>{staged.kind === 'set'
								? 'new portrait staged'
								: 'removal staged'}</Badge
						>
					{:else if override?.drift}
						<Badge tone="warning">drift: Plex moved to a new path, run sync</Badge>
					{:else if override}
						<Badge tone="success">override in place</Badge>
					{/if}
					{#if !actor.path}
						<Badge>no photo in Plex, nothing to override</Badge>
					{/if}
					{#if override?.problem}
						<Badge tone="danger">{override.problem.detail}</Badge>
					{/if}
				</div>
				{#if actor.path}
					<div class="mt-5 flex flex-wrap items-center gap-2">
						<Button onclick={() => (editing = true)}>
							<ImagePlus class="size-4" aria-hidden="true" />
							{override || staged?.kind === 'set'
								? 'Replace portrait'
								: 'Choose portrait'}
						</Button>
						{#if staged}
							<Button variant="secondary" onclick={undo}>Undo</Button>
						{:else if override}
							<Button variant="danger" onclick={remove}>Remove override</Button>
						{/if}
						{#if showsPlex}
							<Tooltip text="Plex's portrait">
								<Avatar
									src={api.cdnImage(actor.path!, 96)}
									alt="Plex's portrait"
									size="md"
								/>
							</Tooltip>
						{/if}
					</div>
					<p class="text-fg-muted mt-2 text-xs">Or drop an image on the portrait.</p>
				{/if}
			</div>
		</div>

		{#each libraries as [library, titles] (library)}
			<section class="mt-8">
				<h2 class="text-fg-muted mb-3 text-sm font-medium">{library}</h2>
				<ul class="flex gap-4 overflow-x-auto pb-4">
					{#each titles as t (t.ratingKey)}
						<li class="w-28 shrink-0">
							<a href="/titles/{t.ratingKey}" use:link class="group block">
								<img
									src={api.posterImage(t.ratingKey, 400)}
									alt=""
									class="bg-surface-raised aspect-2/3 w-28 rounded-md object-cover transition-opacity group-hover:opacity-80"
									loading="lazy"
								/>
								<p class="mt-2 truncate text-sm font-medium" title={t.name}>
									{t.name}
								</p>
								<p class="text-fg-muted text-xs">{t.year ?? ''}</p>
							</a>
						</li>
					{/each}
				</ul>
			</section>
		{:else}
			<p class="text-fg-muted mt-8 text-sm">Nothing in your libraries.</p>
		{/each}

		{#snippet row(label: string, value: string)}
			<div class="flex items-center gap-3 py-2">
				<span class="text-fg-muted w-20 shrink-0 text-xs">{label}</span>
				<span class="min-w-0 flex-1 truncate font-mono text-xs">{value}</span>
				<Button
					variant="ghost"
					size="icon"
					label="Copy {label}"
					onclick={() => copy(value)}
				>
					{#if copied === value}
						<Check class="text-success size-4" aria-hidden="true" />
					{:else}
						<Copy class="size-4" aria-hidden="true" />
					{/if}
				</Button>
			</div>
		{/snippet}
		<Dialog bind:open={details} title="Details">
			<div class="divide-border divide-y">
				{#if actor.tagKey}
					{@render row('person', actor.tagKey)}
				{/if}
				{#if actor.path}
					{@render row('path', actor.path)}
				{/if}
				{#if override}
					{@render row('image', override.image)}
					{#if override.resolved}
						{@render row('resolved', new Date(override.resolved).toLocaleString())}
					{/if}
					{#if override.history?.length}
						{@render row(
							'moved',
							`${override.history.length} time${override.history.length === 1 ? '' : 's'}`
						)}
					{/if}
				{/if}
			</div>
		</Dialog>

		<PortraitDialog
			bind:open={editing}
			actorKey={actor.key}
			actorName={actor.name}
			bind:source
			tmdb={tmdb ? lookup : null}
			onstaged={load}
		/>
	{/if}
</div>
