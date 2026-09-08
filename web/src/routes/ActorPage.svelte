<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';
	import { api, ApiError, type ActorPage } from '$lib/api/client';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Card from '$lib/ui/card/Card.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import { link } from '$lib/router/router.svelte';

	let { key }: { key: string } = $props();

	let page = $state<ActorPage | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		page = null;
		error = null;
		api.actor(key)
			.then((p) => (page = p))
			.catch(
				(e: ApiError) => (error = e.status === 404 ? 'No actor with that key.' : e.message)
			);
	});

	const actor = $derived(page?.actor);
	const override = $derived(page?.override);
	// titles grouped by library, in the order the libraries first appear
	const libraries = $derived.by(() => {
		const groups = new Map<
			string,
			typeof actor extends undefined ? never : NonNullable<typeof actor>['titles']
		>();
		for (const t of actor?.titles ?? []) {
			if (!groups.has(t.library)) groups.set(t.library, []);
			groups.get(t.library)!.push(t);
		}
		return [...groups.entries()];
	});
</script>

<div class="mx-auto w-full max-w-3xl px-4 pb-16 sm:px-6">
	<a
		href="/"
		use:link
		class="text-fg-muted hover:text-fg mb-4 inline-flex items-center gap-1.5 text-sm transition-colors"
	>
		<ArrowLeft class="size-4" aria-hidden="true" />
		Search
	</a>

	{#if error}
		<p class="text-danger">{error}</p>
	{:else if !actor}
		<div class="text-fg-muted flex items-center gap-3"><Spinner /> Loading from Plex…</div>
	{:else}
		<div class="flex flex-col gap-6 sm:flex-row sm:items-start">
			<Avatar src={actor.path ? api.cdnImage(actor.path, 400) : undefined} alt="" size="xl" />
			<div class="min-w-0 flex-1">
				<h1 class="text-3xl font-semibold tracking-tight">{actor.name}</h1>
				<p class="text-fg-muted mt-1">{actor.libraries.join(', ')}</p>
				<dl
					class="text-fg-muted mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 font-mono text-xs"
				>
					{#if actor.tagKey}
						<dt>person</dt>
						<dd class="text-fg">{actor.tagKey}</dd>
					{/if}
					<dt>tag</dt>
					<dd class="text-fg">{actor.key}</dd>
					{#if actor.path}
						<dt>path</dt>
						<dd class="text-fg truncate">{actor.path}</dd>
					{/if}
				</dl>
				<div class="mt-3 flex flex-wrap gap-2">
					{#if override?.drift}
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
			</div>
		</div>

		{#each libraries as [library, titles] (library)}
			<section class="mt-8">
				<h2 class="text-fg-muted mb-3 text-sm font-medium">{library}</h2>
				<ul class="flex gap-4 overflow-x-auto pb-2">
					{#each titles as t (t.ratingKey)}
						<li class="w-28 shrink-0">
							<img
								src={api.posterImage(t.ratingKey, 400)}
								alt=""
								class="bg-surface-raised aspect-[2/3] w-28 rounded-md object-cover"
								loading="lazy"
							/>
							<p class="mt-2 truncate text-sm font-medium" title={t.name}>{t.name}</p>
							<p class="text-fg-muted text-xs">{t.year ?? ''}</p>
						</li>
					{/each}
				</ul>
			</section>
		{:else}
			<p class="text-fg-muted mt-8 text-sm">Nothing in your libraries.</p>
		{/each}

		<div class="mt-6 grid gap-6 sm:grid-cols-2">
			<Card title="In Plex now">
				<div class="flex items-center gap-4">
					<Avatar
						src={actor.path ? api.cdnImage(actor.path, 400) : undefined}
						alt=""
						size="lg"
					/>
					<p class="text-fg-muted text-sm">
						{actor.path
							? 'The portrait Plex serves from its CDN.'
							: 'Plex has no portrait for this person.'}
					</p>
				</div>
			</Card>
			<Card title="Override">
				{#if override}
					<div class="flex items-center gap-4">
						<Avatar src={api.portraitImage(override.image)} alt="" size="lg" />
						<div class="min-w-0 text-sm">
							<p class="text-fg-muted truncate font-mono text-xs">{override.image}</p>
							{#if override.resolved}
								<p class="text-fg-muted mt-1">
									Resolved {new Date(override.resolved).toLocaleString()}
								</p>
							{:else}
								<p class="text-fg-muted mt-1">Not resolved yet. Run sync.</p>
							{/if}
							{#if override.history?.length}
								<p class="text-fg-muted mt-1">
									Moved {override.history.length}× as Plex changed the path.
								</p>
							{/if}
						</div>
					</div>
				{:else}
					<p class="text-fg-muted text-sm">
						No override yet. Uploading a portrait comes next.
					</p>
				{/if}
			</Card>
		</div>
	{/if}
</div>
