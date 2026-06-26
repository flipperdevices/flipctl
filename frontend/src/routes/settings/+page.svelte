<script lang="ts">
	import MenuList from '$lib/components/MenuList.svelte';
	import type { MenuItem } from '$lib/components/MenuList.svelte';

	const items: MenuItem[] = [
		{ id: 'wifi', label: 'WiFi', icon: 'wifi', sub: 'Connected' },
		{ id: 'display', label: 'Display', icon: 'settings' },
		{ id: 'sound', label: 'Sound & Vibro', icon: 'settings' },
		{ id: 'power', label: 'Power', icon: 'battery', sub: '78%' },
		{ id: 'desktop', label: 'Desktop', icon: 'apps' },
		{ id: 'update', label: 'Firmware Update', icon: 'settings' },
		{ id: 'about', label: 'About', icon: 'settings' }
	];

	let selectedIndex = 0;
	let showAbout = false;

	function onSelect(e: CustomEvent<MenuItem>) {
		showAbout = e.detail.id === 'about';
	}
</script>

<div class="screen-page">
	{#if showAbout}
		<div class="about-panel">
			<div class="about-row"><span>Device</span><span>Flipper Zero</span></div>
			<div class="about-row"><span>flipctl</span><span>v0.1.0</span></div>
			<div class="about-row"><span>Firmware</span><span>0.93.1</span></div>
			<div class="about-row"><span>Build</span><span>release</span></div>
			<div class="about-row"><span>Radio</span><span>CC1101</span></div>
			<div class="about-row"><span>NFC</span><span>ST25R3916</span></div>
			<button class="back-btn" on:click={() => (showAbout = false)}>← Back</button>
		</div>
	{:else}
		<MenuList
			{items}
			bind:selectedIndex
			on:select={onSelect}
			on:change={(e) => (selectedIndex = e.detail)}
		/>
	{/if}
</div>

<style>
	.screen-page {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.about-panel {
		flex: 1;
		display: flex;
		flex-direction: column;
		padding: 8px;
		gap: 0;
		animation: page-in 60ms steps(3, end) both;
	}

	@keyframes page-in {
		from { opacity: 0; transform: translateX(4px); }
		to { opacity: 1; transform: translateX(0); }
	}

	.about-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 6px 4px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.12);
		font-size: 7px;
	}

	.about-row span:first-child {
		color: rgba(0, 0, 0, 0.5);
	}

	.about-row span:last-child {
		color: var(--pixel);
	}

	.back-btn {
		margin-top: 12px;
		background: var(--pixel);
		color: var(--lcd);
		border: none;
		font-family: 'Press Start 2P', monospace;
		font-size: 7px;
		padding: 6px 10px;
		cursor: pointer;
		align-self: flex-start;
		letter-spacing: 0.5px;
	}
</style>
