<script lang="ts">
	type PinMode = 'IN' | 'OUT' | 'PWM' | '---';
	interface Pin {
		num: number;
		name: string;
		mode: PinMode;
		state: 0 | 1;
	}

	let pins: Pin[] = [
		{ num: 1, name: '3.3V', mode: '---', state: 1 },
		{ num: 2, name: 'PA7', mode: 'OUT', state: 0 },
		{ num: 3, name: 'PA6', mode: 'IN',  state: 1 },
		{ num: 4, name: 'PA4', mode: 'OUT', state: 0 },
		{ num: 5, name: 'PB3', mode: 'PWM', state: 1 },
		{ num: 6, name: 'PB2', mode: 'OUT', state: 0 },
		{ num: 7, name: 'PC3', mode: 'IN',  state: 0 },
		{ num: 8, name: 'GND', mode: '---', state: 0 }
	];

	let selectedPin = 0;

	function togglePin(i: number) {
		if (pins[i].mode === 'OUT' || pins[i].mode === 'PWM') {
			pins[i] = { ...pins[i], state: pins[i].state === 0 ? 1 : 0 };
			pins = [...pins];
		}
	}
</script>

<div class="screen-page">
	<div class="gpio-header">
		<span class="header-txt">GPIO Pins</span>
		<span class="header-hint">Click OUT/PWM to toggle</span>
	</div>

	<div class="pin-grid">
		{#each pins as pin, i}
			<!-- svelte-ignore a11y-click-events-have-key-events -->
			<div
				class="pin-row"
				class:selected={selectedPin === i}
				class:active={pin.state === 1}
				on:click={() => { selectedPin = i; togglePin(i); }}
				on:mouseenter={() => (selectedPin = i)}
				role="button"
				tabindex="-1"
			>
				<span class="pin-num">P{pin.num}</span>
				<span class="pin-name">{pin.name}</span>
				<span class="pin-mode mode-{pin.mode.toLowerCase()}">{pin.mode}</span>
				<div class="pin-state" class:high={pin.state === 1}>
					<span>{pin.state === 1 ? 'HI' : 'LO'}</span>
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	.screen-page {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.gpio-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 5px 10px;
		background: rgba(0, 0, 0, 0.08);
		border-bottom: 1px solid rgba(0, 0, 0, 0.2);
		flex-shrink: 0;
	}

	.header-txt {
		font-size: 7px;
		text-transform: uppercase;
		color: var(--pixel);
	}

	.header-hint {
		font-size: 5px;
		color: rgba(0, 0, 0, 0.4);
		letter-spacing: 0.3px;
	}

	.pin-grid {
		flex: 1;
		overflow: auto;
	}

	.pin-row {
		display: grid;
		grid-template-columns: 28px 52px 38px 1fr;
		align-items: center;
		gap: 4px;
		padding: 0 8px;
		height: 32px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.1);
		cursor: pointer;
	}

	.pin-row.selected {
		background: var(--selected-bg);
		color: var(--selected-fg);
	}

	.pin-num {
		font-size: 6px;
		color: rgba(0, 0, 0, 0.4);
	}

	.pin-row.selected .pin-num {
		color: rgba(255, 255, 255, 0.5);
	}

	.pin-name {
		font-size: 7px;
		font-family: 'Press Start 2P', monospace;
		letter-spacing: 0.5px;
	}

	.pin-mode {
		font-size: 6px;
		padding: 1px 3px;
		border: 1px solid currentColor;
		text-align: center;
	}

	.mode-in  { color: #555; }
	.mode-out { color: #000; }
	.mode-pwm { color: #000; }
	.mode---- { color: rgba(0,0,0,0.3); }

	.pin-row.selected .pin-mode { color: inherit; }

	.pin-state {
		display: flex;
		justify-content: flex-end;
		align-items: center;
	}

	.pin-state span {
		font-size: 6px;
		padding: 1px 3px;
		background: transparent;
		border: 1px solid rgba(0, 0, 0, 0.3);
	}

	.pin-state.high span {
		background: var(--pixel);
		color: var(--lcd);
		border-color: var(--pixel);
	}

	.pin-row.selected .pin-state.high span {
		background: var(--selected-fg);
		color: var(--selected-bg);
	}
</style>
