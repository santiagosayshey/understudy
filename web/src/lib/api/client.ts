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

export const api = {
	status: () => request<Status>('/api/status'),
	search: (q: string) => request<Actor[]>('/api/actors?q=' + encodeURIComponent(q)),
	refresh: () => request<void>('/api/actors/refresh', { method: 'POST' }),
	cdnImage: (path: string, w = 96) => `/api/images/cdn?w=${w}&path=${encodeURIComponent(path)}`,
};
