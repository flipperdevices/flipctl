import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5174,
		strictPort: true,
		host: '0.0.0.0'
	},
	preview: {
		port: 5174,
		strictPort: true
	}
});
