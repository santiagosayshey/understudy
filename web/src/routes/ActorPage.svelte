<script lang="ts">
	import { ArrowLeft, ArrowRight, Check, Copy, Info, Upload } from '@lucide/svelte';
	import { api, ApiError, type ActorPage } from '$lib/api/client';
	import { discard, stage } from '$lib/changes/changes.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Dialog from '$lib/ui/dialog/Dialog.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import FileInput from '$lib/ui/file-input/FileInput.svelte';
	import { link } from '$lib/router/router.svelte';
	import PortraitDialog from '$lib/editor/PortraitDialog.svelte';

	let { key }: { key: string } = $props();

	let page = $state<ActorPage | null>(null);
	let error = $state<string | null>(null);
	let file = $state<File | null>(null);
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
	const hasOverride = $derived(
		staged?.kind === 'set' || (!!override && staged?.kind !== 'remove')
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
		<div class="flex items-start justify-between gap-4">
			<h1 class="text-3xl font-semibold tracking-tight">{actor.name}</h1>
			<div class="mt-1 flex shrink-0 items-center gap-1">
				<Button
					variant="ghost"
					size="icon"
					label="Details"
					onclick={() => (details = true)}
				>
					<Info class="size-4" aria-hidden="true" />
				</Button>
				<a
					href="/"
					use:link
					class="text-fg-muted hover:text-fg inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-sm transition-colors"
				>
					<ArrowLeft class="size-4" aria-hidden="true" />
					Search
				</a>
			</div>
		</div>
		{#if override?.drift || override?.problem || !actor.path}
			<div class="mt-2 flex flex-wrap gap-2">
				{#if override?.drift}
					<Badge tone="warning">drift: Plex moved to a new path, run sync</Badge>
				{/if}
				{#if override?.problem}
					<Badge tone="danger">{override.problem.detail}</Badge>
				{/if}
				{#if !actor.path}
					<Badge>no photo in Plex, nothing to override</Badge>
				{/if}
			</div>
		{/if}

		<div class="mt-6 flex items-center gap-6">
			<figure class="flex flex-col items-center gap-2">
				<Avatar
					src={actor.path ? api.cdnImage(actor.path, 400) : undefined}
					alt=""
					size="xl"
				/>
				<figcaption class="text-fg-muted text-xs">Plex</figcaption>
			</figure>
			{#if actor.path}
				<ArrowRight class="text-fg-muted size-5 shrink-0" aria-hidden="true" />
				<div
					class="flex flex-col items-center gap-2"
					role="presentation"
					ondragover={(e) => {
						e.preventDefault();
						over = true;
					}}
					ondragleave={() => (over = false)}
					{ondrop}
				>
					{#if hasOverride}
						<div
							class="rounded-full transition-shadow {over
								? 'ring-ring/40 ring-4'
								: ''}"
						>
							<Avatar src={effective} alt="" size="xl" />
						</div>
						<p class="text-fg-muted text-xs">
							{staged?.kind === 'set' ? 'staged' : 'override'}
						</p>
						<div class="flex gap-1">
							{#if staged}
								<Button variant="ghost" size="sm" onclick={undo}>Undo</Button>
							{:else}
								<FileInput
									onfile={choose}
									class="text-fg hover:bg-surface-hover inline-flex h-8 items-center rounded-md px-3 text-sm font-medium transition-colors"
								>
									Replace
								</FileInput>
								<Button variant="ghost" size="sm" onclick={remove}>Remove</Button>
							{/if}
						</div>
					{:else}
						<FileInput
							onfile={choose}
							class="text-fg-muted hover:border-border-strong hover:text-fg flex size-40 flex-col items-center justify-center gap-2 rounded-full border-2 border-dashed text-xs transition-colors {over
								? 'border-ring text-fg'
								: 'border-border'}"
						>
							<Upload class="size-5" aria-hidden="true" />
							<span>drop or choose</span>
						</FileInput>
						{#if staged?.kind === 'remove'}
							<p class="text-fg-muted text-xs">removal staged</p>
							<Button variant="ghost" size="sm" onclick={undo}>Undo</Button>
						{:else}
							<p class="text-fg-muted text-xs">no override</p>
						{/if}
					{/if}
				</div>
			{/if}
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
			{file}
			onstaged={load}
		/>
	{/if}
</div>
