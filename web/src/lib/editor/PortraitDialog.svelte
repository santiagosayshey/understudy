<script lang="ts">
	// Choose a file or one of TMDb's, crop it, stage it. The dialog owns the
	// upload; the actor page only learns that a change was staged.
	import { api, type UploadInfo, type UploadSource } from '$lib/api/client';
	import { stage } from '$lib/changes/changes.svelte';
	import Dialog from '$lib/ui/dialog/Dialog.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Cropper from './Cropper.svelte';

	let {
		open = $bindable(false),
		actorKey,
		actorName,
		source,
		onstaged,
	}: {
		open?: boolean;
		actorKey: string;
		actorName: string;
		source: UploadSource | null;
		onstaged: () => void;
	} = $props();

	let upload = $state<UploadInfo | null>(null);
	let crop = $state({ x: 0, y: 0, size: 0 });
	let error = $state<string | null>(null);
	let busy = $state(false);

	$effect(() => {
		if (!open || !source) return;
		upload = null;
		error = null;
		const held = 'file' in source ? api.upload(source.file) : api.tmdbUpload(source.tmdb);
		held.then((u) => (upload = u)).catch((e: Error) => (error = e.message));
	});

	async function save() {
		if (!upload) return;
		busy = true;
		error = null;
		try {
			await stage({ key: actorKey, kind: 'set', upload: upload.id, crop });
			open = false;
			onstaged();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
</script>

<Dialog bind:open title="Portrait for {actorName}" size="lg">
	{#if error}
		<Notice tone="danger">{error}</Notice>
	{/if}
	{#if upload}
		<Cropper
			src={api.uploadImage(upload.id)}
			width={upload.width}
			height={upload.height}
			face={upload.face ?? null}
			bind:crop
		/>
	{:else if !error}
		<p class="text-fg-muted text-sm">
			{source && 'tmdb' in source ? 'Fetching from TMDb…' : 'Uploading…'}
		</p>
	{/if}
	{#snippet footer()}
		<div class="flex items-center justify-between gap-2">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} loading={busy} disabled={!upload}>Stage</Button>
		</div>
	{/snippet}
</Dialog>
