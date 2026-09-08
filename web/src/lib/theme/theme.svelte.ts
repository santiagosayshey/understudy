// The theme: light or dark. A browser that has never chosen follows the
// system; the first click stores a choice. Applied as a class on <html>,
// which the tokens in app.css switch on.
export type Theme = 'light' | 'dark';

const media = matchMedia('(prefers-color-scheme: dark)');

function initial(): Theme {
	const t = localStorage.getItem('theme');
	if (t === 'light' || t === 'dark') return t;
	return media.matches ? 'dark' : 'light';
}

export const theme = $state({ value: initial() });

function apply() {
	document.documentElement.classList.toggle('dark', theme.value === 'dark');
}

export function setTheme(t: Theme) {
	theme.value = t;
	localStorage.setItem('theme', t);
	apply();
}

media.addEventListener('change', () => {
	if (!localStorage.getItem('theme')) {
		theme.value = media.matches ? 'dark' : 'light';
		apply();
	}
});
