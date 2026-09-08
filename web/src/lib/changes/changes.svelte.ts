// The staged changes, mirrored from the server so the review pill and the
// actor page agree. Refreshed after every stage, discard and apply.
import { api, forgetActors, type Change } from '$lib/api/client';

// applied counts applies, so open pages know to reload after one.
export const pending = $state({ list: [] as Change[], loaded: false, applied: 0 });

export async function refreshChanges() {
	try {
		pending.list = await api.changes();
		pending.loaded = true;
	} catch {
		// the pill just keeps its last count
	}
}

export async function stage(body: Parameters<typeof api.stage>[0]) {
	const c = await api.stage(body);
	forgetActors();
	await refreshChanges();
	return c;
}

export async function discard(key: string) {
	await api.discard(key);
	forgetActors();
	await refreshChanges();
}

export async function apply() {
	const done = await api.apply();
	forgetActors();
	await refreshChanges();
	pending.applied++;
	return done;
}
