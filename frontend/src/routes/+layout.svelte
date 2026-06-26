<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount, onDestroy } from 'svelte';
	import { leftAction, rightAction } from '$lib/stores/actions';
	import StatusBar from '$lib/components/StatusBar.svelte';
	import ActionBar from '$lib/components/ActionBar.svelte';
	import '../app.css';

	$: isRoot = $page.url.pathname === '/';
	$: pageTitle = getTitle($page.url.pathname);

	function getTitle(path: string): string {
		const map: Record<string, string> = {
			'/': 'flipctl',
			'/sub-ghz': 'Sub-GHz',
			'/nfc': 'NFC',
			'/infrared': 'Infrared',
			'/gpio': 'GPIO',
			'/apps': 'Apps',
			'/settings': 'Settings'
		};
		return map[path] ?? 'flipctl';
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.key === 'Escape' || e.key === 'Backspace') && !isRoot) {
			e.preventDefault();
			goto('/');
		}
	}

	$: {
		if (isRoot) {
			leftAction.set(null);
			rightAction.set({ key: '●', label: 'OK' });
		} else {
			leftAction.set({ key: '←', label: 'Back' });
			rightAction.set({ key: '●', label: 'OK' });
		}
	}

	onMount(() => window.addEventListener('keydown', handleKeydown));
	onDestroy(() => window.removeEventListener('keydown', handleKeydown));
</script>

<div class="shell">
	<StatusBar title={pageTitle} />

	<div class="screen">
		{#key $page.url.pathname}
			<div class="page-enter">
				<slot />
			</div>
		{/key}
	</div>

	<ActionBar on:click={() => !isRoot && goto('/')} />
</div>

<style>
	.shell {
		width: 100vw;
		height: 100vh;
		display: flex;
		flex-direction: column;
		background: #0a0a0a;
	}

	.screen {
		flex: 1;
		background: var(--lcd);
		overflow: hidden;
		position: relative;
		/* subtle scanlines for authenticity */
		background-image: repeating-linear-gradient(
			0deg,
			rgba(0, 0, 0, 0.04) 0px,
			rgba(0, 0, 0, 0.04) 1px,
			transparent 1px,
			transparent 2px
		);
	}

	.page-enter {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		animation: page-in 80ms steps(4, end) both;
	}

	@keyframes page-in {
		from {
			opacity: 0;
			transform: translateX(6px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}
</style>
