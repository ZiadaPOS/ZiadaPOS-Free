import { cp, mkdir, rm, stat } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.versions.bun) {
  throw new Error('Run this script with Node.js, not Bun: node ../desktop/scripts/prepare-frontend.mjs');
}

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../../..');
const webRoot = join(repoRoot, 'apps', 'web');
const resourcesRoot = join(repoRoot, 'apps', 'desktop', 'resources');
const standaloneRoot = join(webRoot, '.next', 'standalone');
const bundledWebRoot = join(resourcesRoot, 'web');
const runtimeRoot = join(resourcesRoot, 'runtime');
const nodeName = process.platform === 'win32' ? 'node.exe' : 'node';

console.log('Building the existing Next.js POS UI in standalone mode...');
const build = spawnSync('bun', ['run', 'build'], {
  cwd: webRoot,
  stdio: 'inherit',
  env: {
    ...process.env,
    BUILD_STANDALONE: 'true',
    NEXT_PUBLIC_ZIADAPOS_LOCAL_MODE: '1',
    NEXT_PUBLIC_SENTRY_DISABLED: '1',
    NEXT_TELEMETRY_DISABLED: '1'
  }
});
if (build.error) throw build.error;
if (build.status !== 0) process.exit(build.status ?? 1);

await stat(join(standaloneRoot, 'server.js'));
await rm(bundledWebRoot, { recursive: true, force: true });
await rm(runtimeRoot, { recursive: true, force: true });
await mkdir(dirname(bundledWebRoot), { recursive: true });
await cp(standaloneRoot, bundledWebRoot, { recursive: true });

const staticAssets = join(webRoot, '.next', 'static');
await mkdir(join(bundledWebRoot, '.next'), { recursive: true });
await cp(staticAssets, join(bundledWebRoot, '.next', 'static'), { recursive: true });

const publicAssets = join(webRoot, 'public');
try {
  await stat(publicAssets);
  await cp(publicAssets, join(bundledWebRoot, 'public'), { recursive: true });
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
}

await mkdir(runtimeRoot, { recursive: true });
await cp(process.execPath, join(runtimeRoot, nodeName));
console.log(`Packaged the Next.js UI and ${process.platform}/${process.arch} Node.js runtime.`);
