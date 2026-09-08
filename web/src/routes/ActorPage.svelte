<script lang="ts">
	import { ArrowLeft, Check, Copy, Upload } from '@lucide/svelte';
	import { api, ApiError, type ActorPage } from '$lib/api/client';
	import { discard, stage } from '$lib/changes/changes.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Badge from '$lib/ui/badge/Badge.svelte';
	import Button from '$lib/ui/button/Button.svelte';
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
			<a
				href="/"
				use:link
				class="text-fg-muted hover:text-fg mt-2 inline-flex shrink-0 items-center gap-1.5 text-sm transition-colors"
			>
				<ArrowLeft class="size-4" aria-hidden="true" />
				Search
			</a>
		</div>

		{#snippet fact(label: string, value: string)}
			<div class="flex items-center gap-2">
				<span class="text-fg-muted w-14 shrink-0 text-xs">{label}</span>
				<span class="min-w-0 flex-1 truncate font-mono text-xs">{value}</span>
				<Button
					variant="ghost"
					size="icon"
					label="Copy {label}"
					onclick={() => copy(value)}
				>
					{#if copied === value}
						<Check class="text-success size-3.5" aria-hidden="true" />
					{:else}
						<Copy class="size-3.5" aria-hidden="true" />
					{/if}
				</Button>
			</div>
		{/snippet}

		<div class="mt-6 grid gap-4 sm:grid-cols-2">
			<section
				class="border-border bg-surface shadow-raised flex flex-col rounded-lg border p-5"
			>
				<div class="flex items-center justify-between">
					<h2 class="text-fg-muted text-sm font-medium">Plex</h2>
					{#if override?.drift}
						<Badge tone="warning">path moved</Badge>
					{/if}
				</div>
				<div class="my-5 flex justify-center">
					<Avatar
						src={actor.path ? api.cdnImage(actor.path, 400) : undefined}
						alt=""
						size="xl"
					/>
				</div>
				<div class="mt-auto space-y-1">
					{#if actor.tagKey}
						{@render fact('person', actor.tagKey)}
					{/if}
					{#if actor.path}
						{@render fact('path', actor.path)}
					{:else}
						<p class="text-fg-muted text-xs">
							Plex has no portrait for this person, so nothing is requested and there
							is nothing to override.
						</p>
					{/if}
				</div>
			</section>

			{#if actor.path}
				<section
					class="bg-surface shadow-raised flex flex-col rounded-lg border p-5 transition-colors {hasOverride
						? 'border-border'
						: over
							? 'border-ring border-dashed'
							: 'border-border-strong border-dashed'}"
					role="presentation"
					ondragover={(e) => {
						e.preventDefault();
						over = true;
					}}
					ondragleave={() => (over = false)}
					{ondrop}
				>
					<div class="flex items-center justify-between">
						<h2 class="text-fg-muted text-sm font-medium">Override</h2>
						{#if staged}
							<Badge tone="warning"
								>{staged.kind === 'set' ? 'staged' : 'removal staged'}</Badge
							>
						{:else if override?.problem}
							<Badge tone="danger">problem</Badge>
						{:else if override}
							<Badge tone="success">in place</Badge>
						{/if}
					</div>
					{#if hasOverride}
						<div class="my-5 flex justify-center">
							<Avatar src={effective} alt="" size="xl" />
						</div>
						<div class="mt-auto space-y-1">
							{#if staged?.kind === 'set'}
								<p class="text-fg-muted text-xs">
									Staged {new Date(staged.stagedAt).toLocaleTimeString()}. Applied
									to the configuration when you review.
								</p>
							{:else if override}
								{@render fact('image', override.image)}
								{#if override.resolved}
									<p class="text-fg-muted text-xs">
										Resolved {new Date(
											override.resolved
										).toLocaleString()}{override.history?.length
											? `, moved ${override.history.length}×`
											: ''}.
									</p>
								{:else}
									<p class="text-fg-muted text-xs">Not resolved yet. Run sync.</p>
								{/if}
								{#if override.problem}
									<p class="text-danger text-xs">{override.problem.detail}</p>
								{/if}
							{/if}
							<div class="flex justify-end gap-1 pt-2">
								{#if staged}
									<Button variant="ghost" size="sm" onclick={undo}>Undo</Button>
								{:else}
									<FileInput
										onfile={choose}
										class="text-fg hover:bg-surface-hover inline-flex h-8 items-center rounded-md px-3 text-sm font-medium transition-colors"
									>
										Replace
									</FileInput>
									<Button variant="ghost" size="sm" onclick={remove}
										>Remove</Button
									>
								{/if}
							</div>
						</div>
					{:else}
						<FileInput
							onfile={choose}
							class="text-fg-muted hover:text-fg my-5 flex flex-1 flex-col items-center justify-center gap-3 rounded-md text-center text-sm transition-colors"
						>
							<span
								class="border-border-strong flex size-40 items-center justify-center rounded-full border-2 border-dashed"
							>
								<Upload class="size-6" aria-hidden="true" />
							</span>
							<span>Drop a portrait here or choose a file.</span>
							<span class="text-xs"
								>You crop it to a square next; nothing is written until you review.</span
							>
						</FileInput>
						{#if staged?.kind === 'remove'}
							<div class="mt-auto flex items-center justify-between gap-2">
								<span class="text-fg-muted text-xs"
									>The override will be removed.</span
								>
								<Button variant="ghost" size="sm" onclick={undo}>Undo</Button>
							</div>
						{/if}
					{/if}
				</section>
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

		<PortraitDialog
			bind:open={editing}
			actorKey={actor.key}
			actorName={actor.name}
			{file}
			onstaged={load}
		/>
	{/if}
</div>
