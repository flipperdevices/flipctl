#!/usr/bin/env node
import { readFileSync, mkdirSync, copyFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const appDir = resolve(process.argv[2] ?? '.');
const manifestPath = resolve(appDir, 'flipctl.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const distDir = resolve(root, 'apps');

console.log(`→ Building ${manifest.id}...`);
execSync('npm run build', { cwd: appDir, stdio: 'inherit' });

mkdirSync(distDir, { recursive: true });
copyFileSync(manifestPath, resolve(appDir, 'build', 'flipctl.json'));

const outFile = resolve(distDir, `${manifest.id}.zip`);
execSync(`zip -rq "${outFile}" .`, { cwd: resolve(appDir, 'build'), stdio: 'inherit' });

console.log(`✓ apps/${manifest.id}.zip`);
