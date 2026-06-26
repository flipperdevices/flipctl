<script lang="ts">
	import MenuList from '$lib/components/MenuList.svelte';
	import type { MenuItem } from '$lib/components/MenuList.svelte';

	const items: MenuItem[] = [
		{ id: 'read', label: 'Read', icon: 'nfc' },
		{ id: 'write', label: 'Write', icon: 'nfc' },
		{ id: 'emulate', label: 'Emulate', icon: 'nfc' },
		{ id: 'mfkey32', label: 'Mfkey32', icon: 'nfc', sub: 'Mifare' },
		{ id: 'detect', label: 'Detect Reader', icon: 'nfc' },
		{ id: 'saved', label: 'Saved', icon: 'nfc' }
	];

	let selectedIndex = 0;
	let status = 'Ready';

	function onSelect(e: CustomEvent<MenuItem>) {
		if (e.detail.id === 'read') status = 'Waiting for card...';
		else if (e.detail.id === 'emulate') status = 'Emulating...';
		else status = `[${e.detail.label}]`;
	}
</script>

<div class="screen-page">
	<div class="nfc-banner">
		<div class="nfc-rings">
			<div class="ring ring-1"></div>
			<div class="ring ring-2"></div>
			<div class="ring ring-3"></div>
		</div>
		<span class="status-text">{status}</span>
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

	.nfc-banner {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 5px 10px;
		background: rgba(0, 0, 0, 0.08);
		border-bottom: 1px solid rgba(0, 0, 0, 0.2);
		flex-shrink: 0;
	}

	.nfc-rings {
		position: relative;
		width: 18px;
		height: 14px;
		flex-shrink: 0;
	}

	.ring {
		position: absolute;
		border: 1px solid var(--pixel);
		border-radius: 50%;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
	}

	.ring-1 { width: 4px; height: 4px; }
	.ring-2 { width: 9px; height: 9px; }
	.ring-3 { width: 14px; height: 14px; }

	.status-text {
		font-size: 7px;
		color: var(--pixel);
		letter-spacing: 0.5px;
	}
</style>
