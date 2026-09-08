<script lang="ts">
	// The crop editor: a fixed square viewport over the uploaded image. Drag
	// to pan, wheel or slider to zoom. The crop can never leave the image,
	// and the smallest zoom fills the square, so the result is always a full
	// square. A circle shows what Plex's round avatar reveals; the previews
	// render the crop at the sizes Plex uses. The crop box goes back to the
	// server in source pixels. When the server found a face, the crop opens
	// on it, and a button brings it back after adjusting.
	import Slider from '$lib/ui/slider/Slider.svelte';
	import Button from '$lib/ui/button/Button.svelte';

	let {
		src,
		width: W,
		height: H,
		face = null,
		crop = $bindable({ x: 0, y: 0, size: 0 }),
	}: {
		src: string;
		width: number;
		height: number;
		face?: { x: number; y: number; size: number } | null;
		crop?: { x: number; y: number; size: number };
	} = $props();

	const V = 400;
	let canvas = $state<HTMLCanvasElement | null>(null);
	let small = $state<HTMLCanvasElement | null>(null);
	let tiny = $state<HTMLCanvasElement | null>(null);
	let img: HTMLImageElement | null = null;
	let s = 1;
	let sMin = 1;
	let ox = 0;
	let oy = 0;
	let zoom = $state(1);
	let drag: { x: number; y: number; ox: number; oy: number } | null = null;

	$effect(() => {
		img = new Image();
		img.onload = () => (face ? goTo(face) : fill());
		img.src = src;
	});

	function clamp() {
		s = Math.max(s, sMin);
		ox = Math.min(0, Math.max(V - W * s, ox));
		oy = Math.min(0, Math.max(V - H * s, oy));
	}
	function fill() {
		sMin = V / Math.min(W, H);
		s = sMin;
		ox = (V - W * s) / 2;
		oy = (V - H * s) / 2;
		zoom = 1;
		clamp();
		draw();
	}
	function box() {
		return { x: -ox / s, y: -oy / s, size: V / s };
	}
	function goTo(b: { x: number; y: number; size: number }) {
		sMin = V / Math.min(W, H);
		s = Math.min(4 * sMin, Math.max(sMin, V / b.size));
		ox = V / 2 - (b.x + b.size / 2) * s;
		oy = V / 2 - (b.y + b.size / 2) * s;
		zoom = s / sMin;
		clamp();
		draw();
	}
	function draw() {
		if (!canvas || !img) return;
		const ctx = canvas.getContext('2d')!;
		ctx.clearRect(0, 0, V, V);
		ctx.drawImage(img, ox, oy, W * s, H * s);
		ctx.save();
		ctx.fillStyle = 'rgba(0,0,0,.45)';
		ctx.beginPath();
		ctx.rect(0, 0, V, V);
		ctx.arc(V / 2, V / 2, V / 2, 0, Math.PI * 2, true);
		ctx.fill('evenodd');
		ctx.restore();
		ctx.strokeStyle = 'rgba(255,255,255,.8)';
		ctx.beginPath();
		ctx.arc(V / 2, V / 2, V / 2 - 0.5, 0, Math.PI * 2);
		ctx.stroke();
		const b = box();
		crop = { x: Math.round(b.x), y: Math.round(b.y), size: Math.round(b.size) };
		// The previews are drawn at the display's pixel ratio so they are as
		// sharp as Plex's own avatars, which come from a 360 px image.
		const dpr = window.devicePixelRatio || 1;
		for (const [c, css] of [
			[small, 120],
			[tiny, 56],
		] as const) {
			if (!c) continue;
			const n = Math.round(css * dpr);
			if (c.width !== n) {
				c.width = n;
				c.height = n;
			}
			const pc = c.getContext('2d')!;
			pc.clearRect(0, 0, n, n);
			pc.save();
			pc.beginPath();
			pc.arc(n / 2, n / 2, n / 2, 0, Math.PI * 2);
			pc.clip();
			pc.imageSmoothingQuality = 'high';
			pc.drawImage(img, b.x, b.y, b.size, b.size, 0, 0, n, n);
			pc.restore();
		}
	}
	function zoomTo(z: number) {
		const b = box();
		const cx = b.x + b.size / 2;
		const cy = b.y + b.size / 2;
		s = sMin * z;
		ox = V / 2 - cx * s;
		oy = V / 2 - cy * s;
		clamp();
		draw();
	}
	function onwheel(e: WheelEvent) {
		e.preventDefault();
		zoom = Math.min(4, Math.max(1, (s / sMin) * (e.deltaY < 0 ? 1.06 : 1 / 1.06)));
		zoomTo(zoom);
	}
	function onpointerdown(e: PointerEvent) {
		drag = { x: e.clientX, y: e.clientY, ox, oy };
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}
	function onpointermove(e: PointerEvent) {
		if (!drag) return;
		ox = drag.ox + (e.clientX - drag.x);
		oy = drag.oy + (e.clientY - drag.y);
		clamp();
		draw();
	}
	function onpointerup() {
		drag = null;
	}

	const low = $derived(crop.size > 0 && crop.size < 500);
</script>

<div class="flex flex-col gap-5 sm:flex-row">
	<div>
		<canvas
			bind:this={canvas}
			width={V}
			height={V}
			class="bg-bg block max-w-full cursor-grab touch-none rounded-lg active:cursor-grabbing"
			{onwheel}
			{onpointerdown}
			{onpointermove}
			{onpointerup}
			onpointercancel={onpointerup}
		></canvas>
		<div class="mt-3 flex items-center gap-3">
			<span class="text-fg-muted text-sm">Zoom</span>
			<Slider
				bind:value={zoom}
				min={1}
				max={4}
				step={0.01}
				label="Zoom"
				oninput={() => zoomTo(zoom)}
			/>
			<Button variant="secondary" size="sm" onclick={fill}>Fill</Button>
			{#if face}
				<Button variant="secondary" size="sm" onclick={() => goTo(face)}>Face</Button>
			{/if}
		</div>
	</div>
	<div class="flex flex-col gap-3 text-sm sm:w-48">
		<p class="text-fg-muted">How Plex will show it</p>
		<div class="flex items-end gap-4">
			<canvas bind:this={small} class="bg-surface-raised size-[120px] rounded-full"></canvas>
			<canvas bind:this={tiny} class="bg-surface-raised size-14 rounded-full"></canvas>
		</div>
		<p class="text-fg-muted">
			Drag to move, wheel or slider to zoom. The circle is what the round avatar reveals; the
			square is what gets saved.
		</p>
		<p class="text-fg-muted font-mono text-xs">
			{crop.size}×{crop.size} of {W}×{H}
		</p>
		{#if low}
			<p class="text-warning text-xs">
				Under 500 px in the source; Plex asks for up to 360 px a side.
			</p>
		{/if}
	</div>
</div>
