import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';

// https://vite.dev/config/
export default defineConfig({
	build: {
		outDir: '../internal/web/dist',
		emptyOutDir: true,
	},
	plugins: [tailwindcss(), svelte()],
	resolve: {
		alias: {
			$lib: path.resolve('./src/lib'),
		},
	},
	server: {
		// The Go binary serves the API in production; in development it runs
		// beside Vite, on the address scripts/dev exports.
		proxy: {
			'/api': 'http://' + (process.env.UNDERSTUDY_LISTEN ?? '127.0.0.1:8090'),
		},
	},
});
