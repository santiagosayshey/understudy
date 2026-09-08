<script lang="ts">
	// The first step of choosing a portrait: one of TMDb's, when the server
	// has a key and TMDb knows the person, or a file of your own, dropped or
	// picked. Either way the crop follows.
	import { Upload } from '@lucide/svelte';
	import { api } from '$lib/api/client';
	import Button from '$lib/ui/button/Button.svelte';
	import FileInput from '$lib/ui/file-input/FileInput.svelte';
	import ImageButton from '$lib/ui/image-button/ImageButton.svelte';
	import Notice from '$lib/ui/notice/Notice.svelte';
	import Spinner from '$lib/ui/spinner/Spinner.svelte';
	import type { TmdbLookup } from './tmdb.svelte';

	let {
		tmdb,
		onpick,
		onfile,
	}: {
		/** the lookup the page made, or null without a key */
		tmdb: TmdbLookup | null;
		onpick: (path: string) => void;
		onfile: (file: File) => void;
	} = $props();

	let over = $state(false);
	function ondrop(e: DragEvent) {
		e.preventDefault();
		over = false;
		const f = e.dataTransfer?.files?.[0];
		if (f && f.type.startsWith('image/')) onfile(f);
	}
</script>

<div class="flex flex-col gap-5">
	{#if tmdb}
		{#if tmdb.error}
			<Notice tone="danger">{tmdb.error}</Notice>
		{:else if tmdb.loading}
			<div class="text-fg-muted flex items-center gap-3 text-sm">
				<Spinner /> Asking TMDb…
			</div>
		{:else if !tmdb.person}
			<p class="text-fg-muted text-sm">No one on TMDb by that name.</p>
		{:else}
			<div>
				{#if tmdb.person.profiles.length}
					<ul class="grid grid-cols-3 gap-2 sm:grid-cols-5">
						{#each tmdb.person.profiles as p (p.path)}
							<li>
								<ImageButton
									src={api.tmdbImage(p.path, 400)}
									label="Use this portrait of {tmdb.person.name}"
									onclick={() => onpick(p.path)}
									class="w-full"
								/>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="text-fg-muted text-sm">TMDb has no pictures of {tmdb.person.name}.</p>
				{/if}
				{#if tmdb.others.length}
					<div class="mt-3 flex flex-wrap items-center gap-2">
						<span class="text-fg-muted text-sm">Not {tmdb.person.name}?</span>
						{#each tmdb.others as c (c.id)}
							<Button
								variant="secondary"
								size="sm"
								onclick={() => tmdb?.choose(c.id)}
							>
								{c.name}{c.knownFor.length ? ` · ${c.knownFor[0]}` : ''}
							</Button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	{/if}
	<div
		role="presentation"
		class="rounded-lg transition-shadow {over ? 'ring-ring/40 ring-4' : ''}"
		ondragover={(e) => {
			e.preventDefault();
			over = true;
		}}
		ondragleave={() => (over = false)}
		{ondrop}
	>
		<FileInput
			{onfile}
			class="border-border text-fg-muted hover:border-border-strong hover:bg-surface-hover hover:text-fg flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-4 py-8 text-sm transition-colors"
		>
			<Upload class="size-5" aria-hidden="true" />
			<span>Click to choose an image, or drop one here</span>
		</FileInput>
	</div>
	{#if !tmdb}
		<p class="text-fg-muted text-xs">
			Set UNDERSTUDY_TMDB_KEY to pick from TMDb's portraits here.
		</p>
	{/if}
</div>
