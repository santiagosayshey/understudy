// The actor on TMDb, looked up quietly when the page opens: the likeliest
// person of that name with their portraits, and the others to switch to.
// The page links to the person; the portrait dialog offers the pictures.
import { api, type TmdbMatch, type TmdbPerson } from '$lib/api/client';

export class TmdbLookup {
	person = $state<TmdbPerson | null>(null);
	match = $state<TmdbMatch | null>(null);
	loading = $state(false);
	error = $state<string | null>(null);
	others = $derived((this.match?.candidates ?? []).filter((c) => c.id !== this.person?.id));

	async load(actorKey: string) {
		this.match = null;
		this.person = null;
		this.error = null;
		this.loading = true;
		try {
			const m = await api.tmdb(actorKey);
			this.match = m;
			this.person = m.person;
		} catch (e) {
			this.error = (e as Error).message;
		} finally {
			this.loading = false;
		}
	}

	/** Switches to another person of the same name. */
	async choose(id: number) {
		this.error = null;
		this.loading = true;
		try {
			this.person = await api.tmdbPerson(id);
		} catch (e) {
			this.error = (e as Error).message;
		} finally {
			this.loading = false;
		}
	}
}
