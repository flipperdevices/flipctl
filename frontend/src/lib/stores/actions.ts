import { writable } from 'svelte/store';

export interface ActionHint {
	key: string;
	label: string;
}

export const leftAction = writable<ActionHint | null>(null);
export const rightAction = writable<ActionHint | null>({ key: '●', label: 'OK' });
