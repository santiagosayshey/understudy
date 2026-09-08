<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowRight, Pencil, Trash2 } from '@lucide/svelte';
	import { api, type Applied, type Status } from '$lib/api/client';
	import { pending, refreshChanges, discard, apply } from '$lib/changes/changes.svelte';
	import { route, match, navigate } from '$lib/router/router.svelte';
	import Header from '$lib/ui/header/Header.svelte';
	import Avatar from '$lib/ui/avatar/Avatar.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Drawer from '$lib/ui/drawer/Drawer.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Search from './routes/Search.svelte';
	import ActorPage from './routes/ActorPage.svelte';
	import TitlePage from './routes/TitlePage.svelte';

	let status = $state<Status | null>(null);
	let review = $state(false);
	let applying = $state(false);
	let applied = $state<Applied | null>(null);
	let applyError = $state<string | null>(null);

	onMount(() => {
		refreshChanges();
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
	const title = $derived(match('/titles/:ratingKey', route.path));
	const count = $derived(pending.list.length);

	async function doApply() {
		applying = true;
		applyError = null;
		try {
			applied = await apply();
		} catch (e) {
			applyError = (e as Error).message;
		} finally {
			applying = false;
		}
	}
</script>

<Header version={status?.version} />
<main>
	{#if actor}
		{#key actor.key}
			<ActorPage key={actor.key} />
		{/key}
	{:else if title}
		{#key title.ratingKey}
			<TitlePage ratingKey={title.ratingKey} />
		{/key}
	{:else}
		<Search listing={status?.listing ?? null} />
	{/if}
</main>

{#if count > 0}
	<div class="fixed right-6 bottom-6 z-30">
		<Button size="lg" onclick={() => (review = true)}>
			Review {count}
			{count === 1 ? 'change' : 'changes'}
			<ArrowRight class="size-4" aria-hidden="true" />
		</Button>
	</div>
{/if}

<Drawer bind:open={review} title="Staged changes">
	{#if applied}
		<Notice tone="success">
			Written to the configuration: {[
				...applied.written,
				...applied.removed.map((n) => n + ' removed'),
			].join(', ') || 'nothing'}. Run sync to resolve and clear Plex's cache.
		</Notice>
	{/if}
	{#if applyError}
		<div class="mt-3"><Notice tone="danger">{applyError}</Notice></div>
	{/if}
	{#if count === 0 && !applied}
		<p class="text-fg-muted text-sm">Nothing staged.</p>
	{/if}
	<ul class="divide-border divide-y">
		{#each pending.list as c (c.key)}
			<li class="flex items-center gap-3 py-3">
				{#if c.kind === 'set'}
					<Avatar src={c.path ? api.cdnImage(c.path, 96) : undefined} alt="" size="md" />
					<ArrowRight class="text-fg-muted size-4 shrink-0" aria-hidden="true" />
					<Avatar src={api.changeImage(c.key, c.stagedAt)} alt="" size="md" />
				{:else}
					<Avatar src={c.path ? api.cdnImage(c.path, 96) : undefined} alt="" size="md" />
					<span class="text-fg-muted text-xs">back to Plex's</span>
				{/if}
				<span class="min-w-0 flex-1">
					<span class="block truncate font-medium">{c.name}</span>
					<span class="text-fg-muted block text-xs"
						>{c.kind === 'set' ? 'new portrait' : 'remove override'}</span
					>
				</span>
				<Button
					variant="ghost"
					size="icon"
					label="Edit"
					onclick={() => {
						review = false;
						navigate('/actors/' + c.key);
					}}
				>
					<Pencil class="size-4" aria-hidden="true" />
				</Button>
				<Button variant="ghost" size="icon" label="Discard" onclick={() => discard(c.key)}>
					<Trash2 class="size-4" aria-hidden="true" />
				</Button>
			</li>
		{/each}
	</ul>
	{#snippet footer()}
		<div class="flex items-center justify-between gap-3">
			<p class="text-fg-muted text-xs">
				Writes the configuration file and the portraits directory.
			</p>
			<Button onclick={doApply} loading={applying} disabled={count === 0}>Apply</Button>
		</div>
	{/snippet}
</Drawer>
