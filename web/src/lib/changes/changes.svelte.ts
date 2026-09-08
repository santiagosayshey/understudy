// The staged changes, mirrored from the server so the review pill and the
// actor page agree. Refreshed after every stage, discard and apply.
import { api, forgetActors, type Change } from '$lib/api/client';

// revision goes up on every stage, discard and apply, so an open page
// knows to reload whatever it is showing.
export const pending = $state({ list: [] as Change[], loaded: false, revision: 0 });

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
	await changed();
	return c;
}

export async function discard(key: string) {
	await api.discard(key);
	await changed();
}

export async function apply() {
	const done = await api.apply();
	await changed();
	return done;
}

async function changed() {
	forgetActors();
	await refreshChanges();
	pending.revision++;
}
