<script lang="ts">
	import { goto } from '$app/navigation';
	import MenuList from '$lib/components/MenuList.svelte';
	import PixelIcon from '$lib/components/PixelIcon.svelte';
	import type { IconName } from '$lib/icons';
	import type { MenuItem } from '$lib/components/MenuList.svelte';

	const items: MenuItem[] = [
		{ id: 'sub-ghz', label: 'Sub-GHz', icon: 'sub-ghz' },
		{ id: 'nfc', label: 'NFC', icon: 'nfc' },
		{ id: 'infrared', label: 'Infrared', icon: 'infrared' },
		{ id: 'ibutton', label: 'iButton', icon: 'ibutton' },
		{ id: 'bad-usb', label: 'Bad USB', icon: 'bad-usb' },
		{ id: 'gpio', label: 'GPIO', icon: 'gpio' },
		{ id: 'apps', label: 'Apps', icon: 'apps' },
		{ id: 'settings', label: 'Settings', icon: 'settings' }
	];

	const routable = new Set(['sub-ghz', 'nfc', 'infrared', 'gpio', 'apps', 'settings']);

	let selectedIndex = 0;
	$: selected = items[selectedIndex];

	function onSelect(e: CustomEvent<MenuItem>) {
		const id = e.detail.id;
		if (routable.has(id)) goto(`/${id}`);
	}
</script>

<div class="main-menu">
	<!-- Left panel: menu list -->
	<div class="panel-list">
		<div class="panel-header">Main Menu</div>
		<MenuList
			{items}
			bind:selectedIndex
			on:select={onSelect}
			on:change={(e) => (selectedIndex = e.detail)}
		/>
	</div>

	<!-- Divider -->
	<div class="divider"></div>

	<!-- Right panel: large icon preview -->
	<div class="panel-preview">
		{#key selected.id}
			<div class="preview-inner">
				<PixelIcon
					name={selected.icon}
					size={64}
					large={true}
					color="var(--pixel)"
				/>
				<span class="preview-label">{selected.label}</span>
			</div>
		{/key}
	</div>
</div>

<style>
	.main-menu {
		display: flex;
		flex-direction: row;
		height: 100%;
		width: 100%;
	}

	.panel-list {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		min-width: 0;
	}

	.panel-header {
		font-size: 6px;
		text-transform: uppercase;
		letter-spacing: 1px;
		color: rgba(0, 0, 0, 0.45);
		padding: 5px 8px 4px 14px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.2);
		flex-shrink: 0;
	}

	.divider {
		width: 1px;
		background: rgba(0, 0, 0, 0.25);
		flex-shrink: 0;
	}

	.panel-preview {
		width: 110px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--lcd-dark);
	}

	.preview-inner {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		animation: pop-in 60ms steps(3, end) both;
	}

	.preview-label {
		font-size: 6px;
		text-transform: uppercase;
		color: var(--pixel);
		letter-spacing: 0.5px;
		text-align: center;
		max-width: 80px;
		line-height: 1.6;
	}

	@keyframes pop-in {
		from {
			opacity: 0;
			transform: scale(0.9);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}
</style>
