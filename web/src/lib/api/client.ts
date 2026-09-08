// The editor's API. One function per endpoint, all returning parsed JSON or
// throwing an Error with the server's message.
export type ListingStatus = {
	loaded: boolean;
	loading: boolean;
	error?: string;
	actors: number;
	libraries: string[] | null;
};

export type Status = { version: string; listing: ListingStatus; overrides: number };

export type Actor = {
	key: string;
	name: string;
	path?: string;
	libraries: string[];
	override: boolean;
	drift: boolean;
};

export type Title = { ratingKey: string; name: string; year?: number; library: string };

export type Detail = Actor & { tagKey?: string; titles: Title[] };

export type Override = {
	name: string;
	tagKey?: string;
	image: string;
	path?: string;
	resolved?: string;
	history?: { path: string; until: string }[];
	problem?: { kind: string; detail: string } | null;
	drift?: boolean;
};

export type ActorPage = { actor: Detail; override?: Override };

export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public body?: unknown
	) {
		super(message);
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, init);
	const text = await res.text();
	let body: unknown = text;
	try {
		body = text ? JSON.parse(text) : undefined;
	} catch {
		// not JSON; keep the text
	}
	if (!res.ok) {
		const message =
			body && typeof body === 'object' && 'error' in body
				? String((body as { error: unknown }).error)
				: res.statusText;
		throw new ApiError(message, res.status, body);
	}
	return body as T;
}

// Actor pages fetched ahead of a click, kept for a minute. The search page
// warms this for the rows on screen once typing pauses.
const warmed = new Map<string, { at: number; page: Promise<ActorPage> }>();
const warmFor = 60_000;

function actorPage(key: string): Promise<ActorPage> {
	const hit = warmed.get(key);
	if (hit && Date.now() - hit.at < warmFor) return hit.page;
	const page = request<ActorPage>('/api/actors/' + encodeURIComponent(key));
	warmed.set(key, { at: Date.now(), page });
	page.catch(() => warmed.delete(key));
	return page;
}

/** Fetches the given actors' pages in the background, a few at a time. */
export function prefetchActors(keys: string[], concurrency = 4) {
	const queue = keys.filter((k) => {
		const hit = warmed.get(k);
		return !hit || Date.now() - hit.at >= warmFor;
	});
	const worker = async () => {
		while (queue.length) {
			const key = queue.shift()!;
			await actorPage(key).catch(() => {});
		}
	};
	for (let i = 0; i < Math.min(concurrency, queue.length); i++) worker();
}

/** Drops the cached pages, for after the configuration changed. */
export function forgetActors() {
	warmed.clear();
}

export const api = {
	status: () => request<Status>('/api/status'),
	search: (q: string) =>
		request<{ results: Actor[]; total: number }>('/api/actors?q=' + encodeURIComponent(q)),
	refresh: () => request<void>('/api/actors/refresh', { method: 'POST' }),
	actor: actorPage,
	cdnImage: (path: string, w = 96) => `/api/images/cdn?w=${w}&path=${encodeURIComponent(path)}`,
	posterImage: (ratingKey: string, w = 200) =>
		`/api/images/poster/${encodeURIComponent(ratingKey)}?w=${w}`,
	portraitImage: (image: string) => `/api/images/portrait?image=${encodeURIComponent(image)}`,
};
