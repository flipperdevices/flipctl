<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface NetInterface {
		name: string;
		ip4: string;
		ip6: string | null;
		status: 'UP' | 'DOWN';
		mac: string | null;
		rx: string | null;
		tx: string | null;
		ssid?: string;
	}

	const interfaces: NetInterface[] = [
		{
			name: 'lo',
			ip4: '127.0.0.1/8',
			ip6: '::1/128',
			status: 'UP',
			mac: null,
			rx: null,
			tx: null
		},
		{
			name: 'eth0',
			ip4: '192.168.1.42/24',
			ip6: 'fe80::a00:27ff:fe8d:c04d/64',
			status: 'UP',
			mac: 'aa:bb:cc:dd:ee:ff',
			rx: '1.2 MB',
			tx: '0.8 MB'
		},
		{
			name: 'wlan0',
			ip4: '192.168.0.108/24',
			ip6: null,
			status: 'UP',
			mac: '11:22:33:44:55:66',
			rx: '4.5 MB',
			tx: '1.1 MB',
			ssid: 'HomeNet'
		},
		{
			name: 'docker0',
			ip4: '172.17.0.1/16',
			ip6: null,
			status: 'DOWN',
			mac: '02:42:ab:cd:ef:01',
			rx: '0 B',
			tx: '0 B'
		}
	];

	let selected = 0;
	let activeCount = interfaces.filter((i) => i.status === 'UP').length;

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			selected = (selected - 1 + interfaces.length) % interfaces.length;
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			selected = (selected + 1) % interfaces.length;
		}
	}

	onMount(() => window.addEventListener('keydown', handleKeydown));
	onDestroy(() => window.removeEventListener('keydown', handleKeydown));
</script>

<div class="ifconfig-app">
	<!-- Header bar -->
	<div class="header">
		<div class="header-left">
			<span class="dot"></span>
			<span class="title">ifconfig</span>
		</div>
		<div class="header-right">
			<span class="active-count">{activeCount} UP</span>
		</div>
	</div>

	<!-- Interface list -->
	<div class="iface-list">
		{#each interfaces as iface, i}
			<!-- svelte-ignore a11y-click-events-have-key-events -->
			<div
				class="iface-block"
				class:selected={selected === i}
				on:click={() => (selected = i)}
				on:mouseenter={() => (selected = i)}
				role="button"
				tabindex="-1"
			>
				<!-- Top row: name + status -->
				<div class="iface-row iface-top">
					<div class="iface-name-wrap">
						<span class="cursor">{selected === i ? '>' : ' '}</span>
						<span class="iface-name">{iface.name}</span>
					</div>
					<span class="iface-status" class:up={iface.status === 'UP'}>
						{iface.status}
					</span>
				</div>

				<!-- IP addresses -->
				<div class="iface-row iface-detail">
					<span class="detail-key">IPv4</span>
					<span class="detail-val mono">{iface.ip4}</span>
				</div>

				{#if iface.ip6}
					<div class="iface-row iface-detail">
						<span class="detail-key">IPv6</span>
						<span class="detail-val mono small">{iface.ip6}</span>
					</div>
				{/if}

				{#if iface.mac}
					<div class="iface-row iface-detail">
						<span class="detail-key">MAC</span>
						<span class="detail-val mono">{iface.mac}</span>
					</div>
				{/if}

				{#if iface.ssid}
					<div class="iface-row iface-detail">
						<span class="detail-key">SSID</span>
						<span class="detail-val">{iface.ssid}</span>
					</div>
				{/if}

				{#if iface.rx !== null}
					<div class="iface-row iface-detail">
						<span class="detail-key">RX/TX</span>
						<span class="detail-val">
							<span class="rx">{iface.rx}</span>
							<span class="sep"> / </span>
							<span class="tx">{iface.tx}</span>
						</span>
					</div>
				{/if}
			</div>

			{#if i < interfaces.length - 1}
				<div class="divider"></div>
			{/if}
		{/each}
	</div>

	<!-- Footer hint -->
	<div class="footer">
		<span class="hint">↑↓ navigate</span>
		<span class="hint-right">mock data</span>
	</div>
</div>

<style>
	.ifconfig-app {
		position: fixed;
		inset: 0;
		display: flex;
		flex-direction: column;
		background: var(--lcd);
		overflow: hidden;
	}

	/* Header */
	.header {
		height: 26px;
		background: var(--status-bg);
		color: var(--status-fg);
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 8px;
		flex-shrink: 0;
		border-bottom: 1px solid #333;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.dot {
		width: 5px;
		height: 5px;
		background: var(--accent);
		display: block;
	}

	.title {
		font-size: 7px;
		letter-spacing: 1px;
		text-transform: uppercase;
	}

	.active-count {
		font-size: 6px;
		color: var(--accent);
		letter-spacing: 0.5px;
	}

	/* Interface list */
	.iface-list {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
	}

	.iface-block {
		padding: 5px 8px 5px 6px;
		cursor: pointer;
		transition: none;
	}

	.iface-block.selected {
		background: var(--selected-bg);
		color: var(--selected-fg);
	}

	.iface-row {
		display: flex;
		align-items: baseline;
		gap: 6px;
		line-height: 1.8;
	}

	.iface-top {
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2px;
	}

	.iface-name-wrap {
		display: flex;
		align-items: center;
		gap: 5px;
	}

	.cursor {
		font-size: 8px;
		width: 9px;
		flex-shrink: 0;
	}

	.iface-name {
		font-size: 9px;
		letter-spacing: 0.5px;
		font-weight: bold;
	}

	.iface-status {
		font-size: 6px;
		padding: 1px 4px;
		border: 1px solid currentColor;
		letter-spacing: 0.5px;
		color: rgba(0, 0, 0, 0.4);
	}

	.iface-status.up {
		color: var(--pixel);
	}

	.iface-block.selected .iface-status {
		color: var(--selected-fg);
		opacity: 0.8;
	}

	.iface-detail {
		padding-left: 14px;
	}

	.detail-key {
		font-size: 5px;
		letter-spacing: 0.3px;
		opacity: 0.5;
		min-width: 28px;
		flex-shrink: 0;
		text-transform: uppercase;
	}

	.detail-val {
		font-size: 6px;
		letter-spacing: 0.3px;
	}

	.detail-val.mono {
		font-family: 'Press Start 2P', monospace;
	}

	.detail-val.small {
		font-size: 5px;
	}

	.rx { opacity: 0.9; }
	.sep { opacity: 0.4; }
	.tx { opacity: 0.9; }

	.divider {
		height: 1px;
		background: rgba(0, 0, 0, 0.15);
		margin: 0 6px;
	}

	.iface-block.selected + .divider {
		opacity: 0;
	}

	.divider:has(+ .iface-block.selected) {
		opacity: 0;
	}

	/* Footer */
	.footer {
		height: 22px;
		background: var(--action-bg);
		border-top: 1px solid #222;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 8px;
		flex-shrink: 0;
	}

	.hint {
		font-size: 5px;
		color: var(--action-fg);
		letter-spacing: 0.3px;
		opacity: 0.6;
	}

	.hint-right {
		font-size: 5px;
		color: var(--accent);
		letter-spacing: 0.3px;
		opacity: 0.7;
	}
</style>
