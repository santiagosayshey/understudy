<script lang="ts">
	// Choose a portrait, crop it, stage it. Opened without a source it offers
	// TMDb's portraits and a drop zone; opened with one, a file dropped on the
	// page, it goes straight to the crop. The dialog owns the upload; the
	// actor page only learns that a change was staged.
	import { ArrowLeft } from '@lucide/svelte';
	import { api, type UploadInfo, type UploadSource } from '$lib/api/client';
	import { stage } from '$lib/changes/changes.svelte';
	import Dialog from '$lib/ui/dialog/Dialog.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Cropper from './Cropper.svelte';
	import SourcePicker from './SourcePicker.svelte';
	import type { TmdbLookup } from './tmdb.svelte';

	let {
		open = $bindable(false),
		actorKey,
		actorName,
		source = $bindable(null),
		tmdb,
		onstaged,
	}: {
		open?: boolean;
		actorKey: string;
		actorName: string;
		/** what to crop; null shows the picker first */
		source?: UploadSource | null;
		tmdb: TmdbLookup | null;
		onstaged: () => void;
	} = $props();

	let upload = $state<UploadInfo | null>(null);
	let crop = $state({ x: 0, y: 0, size: 0 });
	let error = $state<string | null>(null);
	let busy = $state(false);

	$effect(() => {
		if (!open) {
			source = null;
			upload = null;
			error = null;
			return;
		}
		if (!source) return;
		upload = null;
		error = null;
		const held = 'file' in source ? api.upload(source.file) : api.tmdbUpload(source.tmdb);
		held.then((u) => (upload = u)).catch((e: Error) => (error = e.message));
	});

	function back() {
		source = null;
		upload = null;
		error = null;
	}

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
	{#if !source}
		<SourcePicker
			{tmdb}
			onpick={(path) => (source = { tmdb: path })}
			onfile={(f) => (source = { file: f })}
		/>
	{:else}
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
				{'tmdb' in source ? 'Fetching from TMDb…' : 'Uploading…'}
			</p>
		{/if}
	{/if}
	{#snippet footer()}
		<div class="flex items-center justify-between gap-2">
			{#if source}
				<Button variant="ghost" onclick={back}>
					<ArrowLeft class="size-4" aria-hidden="true" />
					Back
				</Button>
			{:else}
				<span></span>
			{/if}
			<div class="flex items-center gap-2">
				<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
				{#if source}
					<Button onclick={save} loading={busy} disabled={!upload}>Stage</Button>
				{/if}
			</div>
		</div>
	{/snippet}
</Dialog>
