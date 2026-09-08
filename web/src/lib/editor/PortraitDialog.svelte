<script lang="ts">
	// Choose a file, crop it, stage it. The dialog owns the upload; the
	// actor page only learns that a change was staged.
	import { api, type UploadInfo } from '$lib/api/client';
	import { stage } from '$lib/changes/changes.svelte';
	import Dialog from '$lib/ui/dialog/Dialog.svelte';
	import Button from '$lib/ui/button/Button.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Cropper from './Cropper.svelte';

	let {
		open = $bindable(false),
		actorKey,
		actorName,
		file,
		onstaged,
	}: {
		open?: boolean;
		actorKey: string;
		actorName: string;
		file: File | null;
		onstaged: () => void;
	} = $props();

	let upload = $state<UploadInfo | null>(null);
	let crop = $state({ x: 0, y: 0, size: 0 });
	let error = $state<string | null>(null);
	let busy = $state(false);

	$effect(() => {
		if (!open || !file) return;
		upload = null;
		error = null;
		api.upload(file)
			.then((u) => (upload = u))
			.catch((e: Error) => (error = e.message));
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
			bind:crop
		/>
		<div class="mt-5 flex justify-end gap-2">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} loading={busy}>Stage</Button>
		</div>
	{:else if !error}
		<p class="text-fg-muted text-sm">Uploading…</p>
	{/if}
</Dialog>
