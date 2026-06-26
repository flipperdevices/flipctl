<script lang="ts">
	import MenuList from '$lib/components/MenuList.svelte';
	import type { MenuItem } from '$lib/components/MenuList.svelte';

	const items: MenuItem[] = [
		{ id: 'universal', label: 'Universal Remotes', icon: 'infrared' },
		{ id: 'learn', label: 'Learn New Remote', icon: 'infrared' },
		{ id: 'saved', label: 'Saved Remotes', icon: 'infrared' }
	];

	let selectedIndex = 0;
	let firing = false;

	function onSelect(e: CustomEvent<MenuItem>) {
		if (e.detail.id === 'universal') {
			firing = true;
			setTimeout(() => (firing = false), 500);
		}
	}
</script>

<div class="screen-page">
	<div class="ir-banner">
		<div class="ir-emitter" class:firing>
			<div class="ir-dot"></div>
			<div class="ir-waves">
				{#each [1,2,3] as i}
					<div class="ir-wave" class:firing style="--d:{i * 60}ms"></div>
				{/each}
			</div>
		</div>
		<div class="ir-info">
			<span class="ir-label">Infrared</span>
			<span class="ir-sub">{firing ? 'Transmitting...' : 'Ready'}</span>
		</div>
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

	.ir-banner {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 6px 10px;
		background: rgba(0, 0, 0, 0.08);
		border-bottom: 1px solid rgba(0, 0, 0, 0.2);
		flex-shrink: 0;
	}

	.ir-emitter {
		display: flex;
		align-items: center;
		gap: 3px;
	}

	.ir-dot {
		width: 8px;
		height: 8px;
		background: var(--pixel);
		border-radius: 50%;
		flex-shrink: 0;
	}

	.ir-waves {
		display: flex;
		gap: 2px;
	}

	.ir-wave {
		width: 3px;
		height: 8px;
		background: var(--pixel);
		opacity: 0.2;
	}

	.ir-wave.firing {
		animation: pulse 0.3s steps(2) var(--d) forwards;
		opacity: 1;
	}

	@keyframes pulse {
		0% { opacity: 1; }
		100% { opacity: 0.2; }
	}

	.ir-info {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}

	.ir-label {
		font-size: 7px;
		color: var(--pixel);
	}

	.ir-sub {
		font-size: 6px;
		color: rgba(0, 0, 0, 0.5);
	}
</style>
