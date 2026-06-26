<script lang="ts">
	import { goto } from '$app/navigation';
	import MenuList from '$lib/components/MenuList.svelte';
	import type { MenuItem } from '$lib/components/MenuList.svelte';

	const items: MenuItem[] = [
		{ id: 'freq-analyzer', label: 'Freq Analyzer', icon: 'sub-ghz', sub: '315-928 MHz' },
		{ id: 'read', label: 'Read', icon: 'sub-ghz' },
		{ id: 'read-raw', label: 'Read RAW', icon: 'sub-ghz' },
		{ id: 'add', label: 'Add Manually', icon: 'sub-ghz' },
		{ id: 'saved', label: 'Saved', icon: 'sub-ghz' },
		{ id: 'test', label: 'Test', icon: 'sub-ghz' }
	];

	let selectedIndex = 0;
	let scanning = false;

	function onSelect(e: CustomEvent<MenuItem>) {
		if (e.detail.id === 'freq-analyzer') scanning = !scanning;
	}
</script>

<div class="screen-page">
	<!-- Frequency display banner -->
	<div class="freq-banner">
		<span class="freq-label">Frequency</span>
		<span class="freq-value">{scanning ? '433.920' : '---'} MHz</span>
		{#if scanning}
			<span class="scanning-dot blink">●</span>
		{/if}
	</div>

	<MenuList
		{items}
		bind:selectedIndex
		on:select={onSelect}
		on:change={(e) => (selectedIndex = e.detail)}
	/>
</div>

<style>
	.screen-page {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.freq-banner {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 5px 10px;
		background: rgba(0, 0, 0, 0.08);
		border-bottom: 1px solid rgba(0, 0, 0, 0.2);
		flex-shrink: 0;
	}

	.freq-label {
		font-size: 6px;
		text-transform: uppercase;
		color: rgba(0, 0, 0, 0.5);
		letter-spacing: 0.5px;
	}

	.freq-value {
		font-size: 8px;
		color: var(--pixel);
		letter-spacing: 1px;
	}

	.scanning-dot {
		font-size: 8px;
		color: var(--pixel);
	}

	.blink {
		animation: blink 0.8s steps(1) infinite;
	}

	@keyframes blink {
		0%, 100% { opacity: 1; }
		50% { opacity: 0; }
	}
</style>
