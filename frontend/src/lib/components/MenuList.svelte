<script lang="ts">
	import { createEventDispatcher, onMount, onDestroy } from 'svelte';
	import PixelIcon from './PixelIcon.svelte';
	import type { IconName } from '$lib/icons';

	export interface MenuItem {
		id: string;
		label: string;
		icon: IconName;
		sub?: string;
	}

	export let items: MenuItem[];
	export let selectedIndex: number = 0;

	const dispatch = createEventDispatcher<{ select: MenuItem; change: number }>();

	const VISIBLE = 8;
	let scrollOffset = 0;

	$: {
		if (selectedIndex < scrollOffset) scrollOffset = selectedIndex;
		if (selectedIndex >= scrollOffset + VISIBLE) scrollOffset = selectedIndex - VISIBLE + 1;
	}

	$: visible = items.slice(scrollOffset, scrollOffset + VISIBLE);
	$: showScrollbar = items.length > VISIBLE;
	$: thumbH = Math.round((VISIBLE / items.length) * 100);
	$: thumbTop = Math.round((scrollOffset / items.length) * 100);

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = (selectedIndex - 1 + items.length) % items.length;
			dispatch('change', selectedIndex);
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = (selectedIndex + 1) % items.length;
			dispatch('change', selectedIndex);
		} else if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			dispatch('select', items[selectedIndex]);
		}
	}

	onMount(() => window.addEventListener('keydown', handleKeydown));
	onDestroy(() => window.removeEventListener('keydown', handleKeydown));
</script>

<div class="menu-wrap">
	<ul class="menu-list" role="menu">
		{#each visible as item, vi}
			{@const realIndex = vi + scrollOffset}
			{@const selected = realIndex === selectedIndex}
			<!-- svelte-ignore a11y-click-events-have-key-events -->
			<li
				class="menu-item"
				class:selected
				role="menuitem"
				tabindex="-1"
				on:click={() => {
					selectedIndex = realIndex;
					dispatch('select', item);
				}}
				on:mouseenter={() => {
					selectedIndex = realIndex;
					dispatch('change', selectedIndex);
				}}
			>
				<div class="item-left">
					<span class="cursor">{selected ? '>' : ' '}</span>
					<span class="item-label">{item.label}</span>
					{#if item.sub}
						<span class="item-sub">{item.sub}</span>
					{/if}
				</div>
				<div class="item-icon">
					<PixelIcon
						name={item.icon}
						size={14}
						color={selected ? 'var(--selected-fg)' : 'var(--pixel)'}
					/>
				</div>
			</li>
		{/each}
	</ul>

	{#if showScrollbar}
		<div class="scrollbar-track">
			<div
				class="scrollbar-thumb"
				style="height:{thumbH}%; top:{thumbTop}%"
			></div>
		</div>
	{/if}
</div>

<style>
	.menu-wrap {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		position: relative;
	}

	.menu-list {
		flex: 1;
		list-style: none;
		overflow: hidden;
	}

	.menu-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 6px 0 4px;
		height: 34px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.12);
		cursor: pointer;
		transition: none;
	}

	.menu-item.selected {
		background: var(--selected-bg);
		color: var(--selected-fg);
		border-bottom-color: transparent;
	}

	.item-left {
		display: flex;
		align-items: center;
		gap: 6px;
		overflow: hidden;
	}

	.cursor {
		font-size: 9px;
		width: 10px;
		flex-shrink: 0;
		color: inherit;
	}

	.item-label {
		font-size: 8px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		color: inherit;
	}

	.item-sub {
		font-size: 6px;
		opacity: 0.6;
		color: inherit;
		margin-left: 4px;
	}

	.item-icon {
		flex-shrink: 0;
		display: flex;
		align-items: center;
	}

	.scrollbar-track {
		position: absolute;
		right: 0;
		top: 0;
		bottom: 0;
		width: 3px;
		background: rgba(0, 0, 0, 0.15);
	}

	.scrollbar-thumb {
		position: absolute;
		right: 0;
		width: 3px;
		background: var(--pixel);
	}
</style>
