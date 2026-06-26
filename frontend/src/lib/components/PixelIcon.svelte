<script lang="ts">
	import { ICONS, LARGE_ICONS, type IconName } from '$lib/icons';

	export let name: IconName;
	export let size: number = 16;
	export let large: boolean = false;
	export let color: string = 'currentColor';

	$: grid = large ? (LARGE_ICONS[name] ?? ICONS[name]) : ICONS[name];
	$: rows = grid ?? [];
	$: gridH = rows.length;
	$: gridW = rows[0]?.length ?? 8;
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 {gridW} {gridH}"
	xmlns="http://www.w3.org/2000/svg"
	shape-rendering="crispEdges"
	style="display:block; flex-shrink:0;"
	aria-hidden="true"
>
	{#each rows as row, y}
		{#each row.split('') as cell, x}
			{#if cell === '1'}
				<rect {x} {y} width="1" height="1" fill={color} />
			{/if}
		{/each}
	{/each}
</svg>
