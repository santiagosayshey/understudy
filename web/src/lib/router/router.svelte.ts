// A hand-rolled router for two routes. The path is reactive state; links
// call navigate() and the browser's back button is honoured.
export const route = $state({ path: location.pathname });

export function navigate(path: string) {
	if (path === route.path) return;
	history.pushState(null, '', path);
	route.path = path;
}

addEventListener('popstate', () => {
	route.path = location.pathname;
});

/** Intercepts clicks on same-origin links so they route in the page. */
export function link(node: HTMLAnchorElement) {
	function onClick(e: MouseEvent) {
		if (
			e.defaultPrevented ||
			e.button !== 0 ||
			e.metaKey ||
			e.ctrlKey ||
			e.shiftKey ||
			e.altKey
		)
			return;
		const href = node.getAttribute('href');
		if (!href || !href.startsWith('/')) return;
		e.preventDefault();
		navigate(href);
	}
	node.addEventListener('click', onClick);
	return { destroy: () => node.removeEventListener('click', onClick) };
}

/** Matches /actors/:key style patterns and returns the params, or null. */
export function match(pattern: string, path: string): Record<string, string> | null {
	const a = pattern.split('/');
	const b = path.split('/');
	if (a.length !== b.length) return null;
	const params: Record<string, string> = {};
	for (let i = 0; i < a.length; i++) {
		if (a[i].startsWith(':')) params[a[i].slice(1)] = decodeURIComponent(b[i]);
		else if (a[i] !== b[i]) return null;
	}
	return params;
}
