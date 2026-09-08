<script lang="ts">
	import { ArrowLeft, Upload } from '@lucide/svelte';
	import { api, ApiError, type ActorPage } from '$lib/api/client';
	import { discard, stage } from '$lib/changes/changes.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import FileButton from '$lib/ui/file-button/FileButton.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import { link } from '$lib/router/router.svelte';
	import PortraitDialog from '$lib/editor/PortraitDialog.svelte';

	let { key }: { key: string } = $props();

	let page = $state<ActorPage | null>(null);
	let error = $state<string | null>(null);
	let file = $state<File | null>(null);
	let editing = $state(false);
	let over = $state(false);

	function load() {
		api.actor(key)
			.then((p) => (page = p))
			.catch(
				(e: ApiError) => (error = e.status === 404 ? 'No actor with that key.' : e.message)
			);
	}
	$effect(() => {
		page = null;
		error = null;
		load();
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

	function choose(f: File) {
		file = f;
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
		load();
	}
	async function undo() {
		if (!actor) return;
		await discard(actor.key);
		load();
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
				{#if showsPlex}
					<div class="flex flex-col items-center gap-1">
						<Avatar src={api.cdnImage(actor.path!, 96)} alt="" size="sm" />
						<span class="text-fg-muted text-xs">Plex's</span>
					</div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-start justify-between gap-4">
					<h1 class="text-3xl font-semibold tracking-tight">{actor.name}</h1>
					<a
						href="/"
						use:link
						class="text-fg-muted hover:text-fg mt-2 inline-flex shrink-0 items-center gap-1.5 text-sm transition-colors"
					>
						<ArrowLeft class="size-4" aria-hidden="true" />
						Search
					</a>
				</div>
				<p class="text-fg-muted mt-1">{actor.libraries.join(', ')}</p>
				<dl
					class="text-fg-muted mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 font-mono text-xs"
				>
					{#if actor.tagKey}
						<dt>person</dt>
						<dd class="text-fg">{actor.tagKey}</dd>
					{/if}
					{#if actor.path}
						<dt>path</dt>
						<dd class="text-fg truncate">{actor.path}</dd>
					{/if}
				</dl>
				<div class="mt-3 flex flex-wrap gap-2">
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
					<div class="mt-5 flex flex-wrap gap-2">
						<FileButton onfile={choose}>
							<Upload class="size-4" aria-hidden="true" />
							{override || staged?.kind === 'set'
								? 'Replace portrait'
								: 'Choose portrait'}
						</FileButton>
						{#if staged}
							<Button variant="secondary" onclick={undo}>Undo</Button>
						{:else if override}
							<Button variant="danger" onclick={remove}>Remove override</Button>
						{/if}
					</div>
					<p class="text-fg-muted mt-2 text-xs">Or drop an image on the portrait.</p>
				{/if}
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

		<PortraitDialog
			bind:open={editing}
			actorKey={actor.key}
			actorName={actor.name}
			{file}
			onstaged={load}
		/>
	{/if}
</div>
